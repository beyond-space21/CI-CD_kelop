package videos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	Users "hifi/Events/Users"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	blake3 "lukechampine.com/blake3"

	Search "hifi/Events/Search"
	Auth "hifi/Services/Auth"
	Mdb "hifi/Services/Mdb"
	storage "hifi/Services/Storage"
	Utils "hifi/Utils"
)

var View func(ctx context.Context, auth bool, claims *Auth.Token, videoID string) error

// nullStringToPtr converts sql.NullString to *string (nil if NULL, pointer to value if not)
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func Handle(req chi.Router) {
	req.Post("/upload", Upload)
	req.Delete("/{videoID}", Delete)
	req.Get("/{videoID}", GetVideo)
	req.Get("/list", ListVideo)
	req.Post("/upload/ack/{videoID}", UploadACK)
	req.Get("/list/self", ListVideoSelf)
	req.Get("/list/following", ListVideoFollowing)
	req.Get("/list/{username}", ListVideoByUsername)
}

func Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user
	var user Users.User
	var bioNull, emailNull sql.NullString
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT id, uid, username, name, role, profile_picture, bio, email, 
			followers, following, total_streams, total_videos, created_at, updated_at
		FROM users WHERE uid = $1`,
		claims.UID,
	).Scan(
		&user.ID, &user.UID, &user.Username, &user.Name, &user.Role,
		&user.ProfilePicture, &bioNull, &emailNull, &user.Followers, &user.Following,
		&user.TotalStreams, &user.TotalVideos, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("Upload: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return
	}
	user.Bio = nullStringToPtr(bioNull)
	user.Email = nullStringToPtr(emailNull)

	videoID := fmt.Sprintf("%x", blake3.Sum256([]byte(claims.UID+time.Now().Format(time.RFC3339)+uuid.New().String())))

	var video Videos
	err = json.NewDecoder(r.Body).Decode(&video)
	if err != nil {
		log.Printf("Upload: failed to decode video: %v", err)
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Failed to decode video")
		return
	}
	video.VideoID = videoID
	video.VideoURL = "videos/" + videoID
	video.VideoThumbnail = "thumbnails/videos/" + videoID + ".jpg"
	video.UserUID = claims.UID
	video.UserUsername = user.Username
	video.CreatedAt = time.Now()
	video.UpdatedAt = video.CreatedAt

	// Insert into video_on_upload
	_, err = Mdb.DB.ExecContext(ctx,
		`INSERT INTO video_on_upload (video_id, video_url, video_thumbnail, video_title, video_description, 
			video_tags, video_views, video_upvotes, video_downvotes, video_comments, user_uid, user_username, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		video.VideoID, video.VideoURL, video.VideoThumbnail, video.VideoTitle, video.VideoDescription,
		video.VideoTags, video.VideoViews, video.VideoUpvotes, video.VideoDownvotes,
		video.VideoComments, video.UserUID, video.UserUsername, video.CreatedAt, video.UpdatedAt,
	)
	if err != nil {
		log.Printf("Upload: failed to insert video on upload: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to insert video on upload")
		return
	}

	gatewayURL, err := storage.GeneratePresignedUploadURL(video.VideoURL, 20*time.Minute)
	if err != nil {
		log.Printf("Upload: failed to generate presigned upload URL: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to generate presigned upload URL")
		return
	}

	gatewayURL_thumbnail, err := storage.GeneratePresignedUploadURL(video.VideoThumbnail, 20*time.Minute)
	if err != nil {
		log.Printf("Upload: failed to generate presigned upload URL for thumbnail: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to generate presigned upload URL for thumbnail")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"message":               "bridge created",
		"bridge_id":             videoID,
		"gateway_url":           gatewayURL,
		"gateway_url_thumbnail": gatewayURL_thumbnail,
	})
}

func UploadACK(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	videoID := chi.URLParam(r, "videoID")
	if videoID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Video ID is required")
		return
	}

	video_obj_key := "videos/" + videoID
	thumbnail_obj_key := "thumbnails/videos/" + videoID + ".jpg"

	video_exists, err := storage.IsFileExists(video_obj_key)
	if err != nil || !video_exists {
		log.Printf("UploadACK: video file not found: %v", err)
		Utils.SendErrorResponse(w, http.StatusNotFound, "Video file not found")
		return
	}

	thumbnail_exists, err := storage.IsFileExists(thumbnail_obj_key)
	if err != nil || !thumbnail_exists {
		log.Printf("UploadACK: thumbnail file not found: %v", err)
		Utils.SendErrorResponse(w, http.StatusNotFound, "Thumbnail file not found")
		return
	}

	// Get video from video_on_upload
	var temp_video Videos
	err = Mdb.DB.QueryRowContext(ctx,
		`SELECT id, video_id, video_url, video_thumbnail, video_title, video_description, 
			video_tags, video_views, video_upvotes, video_downvotes, video_comments, 
			user_uid, user_username, created_at, updated_at
		FROM video_on_upload WHERE video_id = $1`,
		videoID,
	).Scan(
		&temp_video.ID, &temp_video.VideoID, &temp_video.VideoURL, &temp_video.VideoThumbnail,
		&temp_video.VideoTitle, &temp_video.VideoDescription, &temp_video.VideoTags,
		&temp_video.VideoViews, &temp_video.VideoUpvotes, &temp_video.VideoDownvotes,
		&temp_video.VideoComments, &temp_video.UserUID, &temp_video.UserUsername,
		&temp_video.CreatedAt, &temp_video.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "Video not found")
		} else {
			log.Printf("UploadACK: failed to fetch video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch video")
		}
		return
	}

	if temp_video.UserUID != claims.UID {
		Utils.SendErrorResponse(w, http.StatusForbidden, "Forbidden: you do not own this video")
		return
	}

	// Delete from video_on_upload
	_, err = Mdb.DB.ExecContext(ctx, "DELETE FROM video_on_upload WHERE video_id = $1", videoID)
	if err != nil {
		log.Printf("UploadACK: failed to delete video on upload: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete video on upload")
		return
	}

	temp_video.UpdatedAt = time.Now()
	temp_video.VideoURL = video_obj_key

	// Insert into videos
	_, err = Mdb.DB.ExecContext(ctx,
		`INSERT INTO videos (video_id, video_url, video_thumbnail, video_title, video_description, 
			video_tags, video_views, video_upvotes, video_downvotes, video_comments, user_uid, user_username, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		temp_video.VideoID, temp_video.VideoURL, temp_video.VideoThumbnail, temp_video.VideoTitle,
		temp_video.VideoDescription, temp_video.VideoTags, temp_video.VideoViews,
		temp_video.VideoUpvotes, temp_video.VideoDownvotes, temp_video.VideoComments,
		temp_video.UserUID, temp_video.UserUsername, temp_video.CreatedAt, temp_video.UpdatedAt,
	)
	if err != nil {
		log.Printf("UploadACK: failed to insert video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to insert video")
		return
	}

	// Update user total_videos
	_, err = Mdb.DB.ExecContext(ctx, "UPDATE users SET total_videos = total_videos + 1 WHERE uid = $1", temp_video.UserUID)
	if err != nil {
		log.Printf("UploadACK: failed to update user total_videos: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Index video in Elasticsearch (non-blocking, log errors but don't fail upload)
	go func() {
		esCtx := context.Background()
		// Convert StringArray to []string for Elasticsearch
		tags := make([]string, len(temp_video.VideoTags))
		for i, tag := range temp_video.VideoTags {
			tags[i] = tag
		}
		if err := Search.IndexVideo(esCtx, temp_video.VideoID, temp_video.VideoTitle, temp_video.VideoDescription, tags, temp_video.UserUsername); err != nil {
			log.Printf("UploadACK: failed to index video in Elasticsearch: %v", err)
		}
	}()

	Utils.SendSuccessResponse(w, map[string]interface{}{"message": "video uploaded"})
}

func Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	videoID := chi.URLParam(r, "videoID")
	if videoID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Video ID is required")
		return
	}

	// Get video
	var video Videos
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT id, video_id, video_url, video_thumbnail, video_title, video_description, 
			video_tags, video_views, video_upvotes, video_downvotes, video_comments, 
			user_uid, user_username, created_at, updated_at
		FROM videos WHERE video_id = $1`,
		videoID,
	).Scan(
		&video.ID, &video.VideoID, &video.VideoURL, &video.VideoThumbnail,
		&video.VideoTitle, &video.VideoDescription, &video.VideoTags,
		&video.VideoViews, &video.VideoUpvotes, &video.VideoDownvotes,
		&video.VideoComments, &video.UserUID, &video.UserUsername,
		&video.CreatedAt, &video.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "Video not found")
		} else {
			log.Printf("Delete: failed to fetch video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch video")
		}
		return
	}

	if video.UserUID != claims.UID {
		Utils.SendErrorResponse(w, http.StatusForbidden, "Forbidden: you do not own this video")
		return
	}

	// Delete video and thumbnail files from R2 storage
	// Use the stored paths from the database
	video_obj_key := video.VideoURL
	thumbnail_obj_key := video.VideoThumbnail

	log.Printf("Delete: Video URL from DB: %s", video_obj_key)
	log.Printf("Delete: Thumbnail URL from DB: %s", thumbnail_obj_key)

	// Verify files exist before attempting deletion
	videoExists, _ := storage.IsFileExists(video_obj_key)
	thumbnailExists, _ := storage.IsFileExists(thumbnail_obj_key)
	log.Printf("Delete: Video exists in R2: %v, Thumbnail exists in R2: %v", videoExists, thumbnailExists)

	// Delete video file from storage
	if videoExists {
		if err := storage.DeleteFile(ctx, video_obj_key); err != nil {
			log.Printf("Delete: CRITICAL ERROR - failed to delete video file from storage (%s): %v", video_obj_key, err)
			// Continue with deletion even if storage cleanup fails
		} else {
			// Verify deletion by checking if file still exists
			exists, checkErr := storage.IsFileExists(video_obj_key)
			if checkErr == nil && exists {
				log.Printf("Delete: CRITICAL WARNING - video file still exists after deletion attempt: %s", video_obj_key)
			} else {
				log.Printf("Delete: successfully deleted and verified video file: %s", video_obj_key)
			}
		}
	} else {
		log.Printf("Delete: video file does not exist in R2, skipping deletion: %s", video_obj_key)
	}

	// Delete thumbnail file from storage
	if thumbnailExists {
		if err := storage.DeleteFile(ctx, thumbnail_obj_key); err != nil {
			log.Printf("Delete: CRITICAL ERROR - failed to delete thumbnail file from storage (%s): %v", thumbnail_obj_key, err)
			// Continue with deletion even if storage cleanup fails
		} else {
			// Verify deletion by checking if file still exists
			exists, checkErr := storage.IsFileExists(thumbnail_obj_key)
			if checkErr == nil && exists {
				log.Printf("Delete: CRITICAL WARNING - thumbnail file still exists after deletion attempt: %s", thumbnail_obj_key)
			} else {
				log.Printf("Delete: successfully deleted and verified thumbnail file: %s", thumbnail_obj_key)
			}
		}
	} else {
		log.Printf("Delete: thumbnail file does not exist in R2, skipping deletion: %s", thumbnail_obj_key)
	}

	// Delete video from database
	// Foreign key CASCADE will automatically delete:
	// - All upvotes on this video
	// - All downvotes on this video
	// - All comments on this video (which triggers deletion of all replies to those comments)
	// - All views of this video
	_, err = Mdb.DB.ExecContext(ctx, "DELETE FROM videos WHERE video_id = $1", videoID)
	if err != nil {
		log.Printf("Delete: failed to delete video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete video")
		return
	}

	// Update user total_videos
	_, err = Mdb.DB.ExecContext(ctx, "UPDATE users SET total_videos = total_videos - 1 WHERE uid = $1", video.UserUID)
	if err != nil {
		// Log but don't fail - user update is non-critical
		log.Printf("Delete: warning - failed to update user total_videos: %v", err)
	}

	// Delete video from Elasticsearch (non-blocking, log errors but don't fail deletion)
	go func() {
		esCtx := context.Background()
		if err := Search.DeleteVideo(esCtx, videoID); err != nil {
			log.Printf("Delete: failed to delete video from Elasticsearch: %v", err)
		}
	}()

	Utils.SendSuccessResponse(w, map[string]interface{}{"message": "video deleted"})
}

func GetVideo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	videoID := chi.URLParam(r, "videoID")
	if videoID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Video ID is required")
		return
	}

	upvoted := false
	downvoted := false
	following := false

	claims, auth := Auth.GetClaims(r)
	putViewErr := View(ctx, auth, claims, videoID)

	// Get video
	var video Videos
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT id, video_id, video_url, video_thumbnail, video_title, video_description, 
			video_tags, video_views, video_upvotes, video_downvotes, video_comments, 
			user_uid, user_username, created_at, updated_at
		FROM videos WHERE video_id = $1`,
		videoID,
	).Scan(
		&video.ID, &video.VideoID, &video.VideoURL, &video.VideoThumbnail,
		&video.VideoTitle, &video.VideoDescription, &video.VideoTags,
		&video.VideoViews, &video.VideoUpvotes, &video.VideoDownvotes,
		&video.VideoComments, &video.UserUID, &video.UserUsername,
		&video.CreatedAt, &video.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "Video not found")
		} else {
			log.Printf("GetVideo: failed to fetch video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch video")
		}
		return
	}

	// Optimized: Check upvoted, downvoted, and following in a single query
	if auth {
		var hasUpvote, hasDownvote, hasFollow bool
		err := Mdb.DB.QueryRowContext(ctx,
			`SELECT 
				EXISTS(SELECT 1 FROM upvotes WHERE upvoted_by = $1 AND upvoted_to = $2) as upvoted,
				EXISTS(SELECT 1 FROM downvotes WHERE downvoted_by = $1 AND downvoted_to = $2) as downvoted,
				EXISTS(SELECT 1 FROM followers WHERE followed_by = $1 AND followed_to = $3) as following`,
			claims.UID, videoID, video.UserUID,
		).Scan(&hasUpvote, &hasDownvote, &hasFollow)
		if err == nil {
			upvoted = hasUpvote
			downvoted = hasDownvote
			following = hasFollow
		}
	}

	// Use Workers URL for video access
	// Note: Client must include header "x-api-key: SECRET_KEY" when requesting the video
	videoURL := fmt.Sprintf("https://black-paper-83cf.hiffi.workers.dev/videos/%s", videoID)

	response := map[string]interface{}{
		"video_url":     videoURL,
		"user_username": video.UserUsername,
		"upvoted":       upvoted,
		"downvoted":     downvoted,
		"following":     following,
	}

	if putViewErr != nil {
		response["put_view_error"] = putViewErr.Error()
	}

	Utils.SendSuccessResponse(w, response)
}

// Constants for pagination
const (
	DefaultVideoPageLimit = 20
	MaxVideoPageLimit     = 100
)

// ListVideo lists all videos with deterministic random pagination (stable shuffle)
// Query params: ?limit=20&offset=0&seed=optional_seed
func ListVideo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if user is authenticated (optional for this endpoint)
	claims, auth := Auth.GetClaims(r)

	// Parse pagination parameters from query string
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	seedStr := r.URL.Query().Get("seed")

	limit := DefaultVideoPageLimit
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= MaxVideoPageLimit {
				limit = l
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Use provided seed or default seed for deterministic random pagination (stable shuffle)
	// Uses hash-based ordering with the specified seed for consistent pseudo-random order
	// This ensures the same "random" order across all pagination requests with the same seed
	seed := seedStr
	if seed == "" {
		seed = "hifi_videos_shuffle_2024" // Default seed for stable shuffle
	}

	// Optimized: Use LEFT JOIN to get following status and user profile_picture in a single query
	// This eliminates the need for a separate query and array collection
	var query string
	var args []interface{}
	if auth {
		query = `SELECT 
			v.id, v.video_id, v.video_url, v.video_thumbnail, v.video_title, v.video_description, 
			v.video_tags, v.video_views, v.video_upvotes, v.video_downvotes, v.video_comments, 
			v.user_uid, v.user_username, v.created_at, v.updated_at,
			u.profile_picture,
			CASE WHEN f.followed_by IS NOT NULL THEN true ELSE false END as following
		FROM videos v
		LEFT JOIN users u ON v.user_uid = u.uid
		LEFT JOIN followers f ON f.followed_by = $1 AND f.followed_to = v.user_uid
		ORDER BY hashtext(v.id::text || $2)
		LIMIT $3 OFFSET $4`
		args = []interface{}{claims.UID, seed, limit, offset}
	} else {
		query = `SELECT 
			v.id, v.video_id, v.video_url, v.video_thumbnail, v.video_title, v.video_description, 
			v.video_tags, v.video_views, v.video_upvotes, v.video_downvotes, v.video_comments, 
			v.user_uid, v.user_username, v.created_at, v.updated_at,
			u.profile_picture,
			false as following
		FROM videos v
		LEFT JOIN users u ON v.user_uid = u.uid
		ORDER BY hashtext(v.id::text || $1)
		LIMIT $2 OFFSET $3`
		args = []interface{}{seed, limit, offset}
	}

	rows, err := Mdb.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("ListVideo: failed to query videos: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
		return
	}
	defer rows.Close()

	var videos []Videos
	var followingMap map[string]bool
	var profilePictureMap map[string]string
	if auth {
		followingMap = make(map[string]bool)
	}
	profilePictureMap = make(map[string]string)
	for rows.Next() {
		var video Videos
		var isFollowing bool
		var profilePicture string
		err := rows.Scan(
			&video.ID, &video.VideoID, &video.VideoURL, &video.VideoThumbnail,
			&video.VideoTitle, &video.VideoDescription, &video.VideoTags,
			&video.VideoViews, &video.VideoUpvotes, &video.VideoDownvotes,
			&video.VideoComments, &video.UserUID, &video.UserUsername,
			&video.CreatedAt, &video.UpdatedAt, &profilePicture, &isFollowing,
		)
		if err != nil {
			log.Printf("ListVideo: failed to scan video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
			return
		}
		videos = append(videos, video)
		profilePictureMap[video.UserUID] = profilePicture
		if auth {
			followingMap[video.UserUID] = isFollowing
		}
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListVideo: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate videos")
		return
	}

	// Build response with following status and profile picture for each video
	type VideoWithFollowing struct {
		Video          Videos `json:"video"`
		Following      bool   `json:"following"`
		ProfilePicture string `json:"profile_picture"`
	}

	videosWithFollowing := make([]VideoWithFollowing, len(videos))
	for i, video := range videos {
		videosWithFollowing[i] = VideoWithFollowing{
			Video:          video,
			Following:      followingMap[video.UserUID],
			ProfilePicture: profilePictureMap[video.UserUID],
		}
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"videos": videosWithFollowing,
		"limit":  limit,
		"offset": offset,
		"count":  len(videos),
		"seed":   seed,
	})
}

// ListVideoSelf lists authenticated user's videos with deterministic random pagination (stable shuffle)
// Query params: ?limit=20&offset=0&seed=optional_seed
func ListVideoSelf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse pagination parameters from query string
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	seedStr := r.URL.Query().Get("seed")

	limit := DefaultVideoPageLimit
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= MaxVideoPageLimit {
				limit = l
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Use provided seed or default seed for deterministic random pagination (stable shuffle)
	// Uses hash-based ordering with the specified seed for consistent pseudo-random order
	// This ensures the same "random" order across all pagination requests with the same seed
	seed := seedStr
	if seed == "" {
		seed = "hifi_videos_self_shuffle_2024" // Default seed for stable shuffle
	}

	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT id, video_id, video_url, video_thumbnail, video_title, video_description, 
			video_tags, video_views, video_upvotes, video_downvotes, video_comments, 
			user_uid, user_username, created_at, updated_at
		FROM videos WHERE user_uid = $1
		ORDER BY hashtext(id::text || $2)
		LIMIT $3 OFFSET $4`,
		claims.UID, seed, limit, offset,
	)
	if err != nil {
		log.Printf("ListVideoSelf: failed to query videos: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
		return
	}
	defer rows.Close()

	var videos []Videos
	for rows.Next() {
		var video Videos
		err := rows.Scan(
			&video.ID, &video.VideoID, &video.VideoURL, &video.VideoThumbnail,
			&video.VideoTitle, &video.VideoDescription, &video.VideoTags,
			&video.VideoViews, &video.VideoUpvotes, &video.VideoDownvotes,
			&video.VideoComments, &video.UserUID, &video.UserUsername,
			&video.CreatedAt, &video.UpdatedAt,
		)
		if err != nil {
			log.Printf("ListVideoSelf: failed to scan video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
			return
		}
		videos = append(videos, video)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListVideoSelf: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate videos")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"videos": videos,
		"limit":  limit,
		"offset": offset,
		"count":  len(videos),
		"seed":   seed,
	})
}

// ListVideoFollowing lists videos from users that the authenticated user follows
// Query params: ?limit=20&offset=0
// Ordered by created_at DESC (newest first)
func ListVideoFollowing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse pagination parameters from query string
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := DefaultVideoPageLimit
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= MaxVideoPageLimit {
				limit = l
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Query videos from users that the authenticated user follows
	// Uses INNER JOIN for efficiency - only returns videos from followed users
	// Ordered by created_at DESC (newest first)
	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT v.id, v.video_id, v.video_url, v.video_thumbnail, v.video_title, v.video_description, 
			v.video_tags, v.video_views, v.video_upvotes, v.video_downvotes, v.video_comments, 
			v.user_uid, v.user_username, v.created_at, v.updated_at
		FROM videos v
		INNER JOIN followers f ON v.user_uid = f.followed_to
		WHERE f.followed_by = $1
		ORDER BY v.created_at DESC
		LIMIT $2 OFFSET $3`,
		claims.UID, limit, offset,
	)
	if err != nil {
		log.Printf("ListVideoFollowing: failed to query videos: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
		return
	}
	defer rows.Close()

	var videos []Videos
	for rows.Next() {
		var video Videos
		err := rows.Scan(
			&video.ID, &video.VideoID, &video.VideoURL, &video.VideoThumbnail,
			&video.VideoTitle, &video.VideoDescription, &video.VideoTags,
			&video.VideoViews, &video.VideoUpvotes, &video.VideoDownvotes,
			&video.VideoComments, &video.UserUID, &video.UserUsername,
			&video.CreatedAt, &video.UpdatedAt,
		)
		if err != nil {
			log.Printf("ListVideoFollowing: failed to scan video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
			return
		}
		videos = append(videos, video)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListVideoFollowing: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate videos")
		return
	}

	// Build response with following status (always true since we filtered by followed users)
	type VideoWithFollowing struct {
		Video     Videos `json:"video"`
		Following bool   `json:"following"`
	}

	videosWithFollowing := make([]VideoWithFollowing, len(videos))
	for i, video := range videos {
		videosWithFollowing[i] = VideoWithFollowing{
			Video:     video,
			Following: true, // Always true since we only show videos from followed users
		}
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"videos": videosWithFollowing,
		"limit":  limit,
		"offset": offset,
		"count":  len(videos),
	})
}

// ListVideoByUsername lists videos uploaded by a specific user ordered by timestamp (newest first)
// Query params: ?limit=20&offset=0
func ListVideoByUsername(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if user is authenticated (optional for this endpoint)
	claims, auth := Auth.GetClaims(r)

	// Get username from URL parameter
	username := chi.URLParam(r, "username")
	if username == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Normalize username
	username = strings.ToLower(strings.TrimSpace(username))

	// Parse pagination parameters from query string
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := DefaultVideoPageLimit
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= MaxVideoPageLimit {
				limit = l
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT v.id, v.video_id, v.video_url, v.video_thumbnail, v.video_title, v.video_description, 
			v.video_tags, v.video_views, v.video_upvotes, v.video_downvotes, v.video_comments, 
			v.user_uid, v.user_username, v.created_at, v.updated_at
		FROM videos v
		INNER JOIN users u ON v.user_uid = u.uid
		WHERE u.username = $1
		ORDER BY v.created_at DESC
		LIMIT $2 OFFSET $3`,
		username, limit, offset,
	)
	if err != nil {
		log.Printf("ListVideoByUsername: failed to query videos: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
		return
	}
	defer rows.Close()

	var videos []Videos
	var videoOwnerUID string
	for rows.Next() {
		var video Videos
		err := rows.Scan(
			&video.ID, &video.VideoID, &video.VideoURL, &video.VideoThumbnail,
			&video.VideoTitle, &video.VideoDescription, &video.VideoTags,
			&video.VideoViews, &video.VideoUpvotes, &video.VideoDownvotes,
			&video.VideoComments, &video.UserUID, &video.UserUsername,
			&video.CreatedAt, &video.UpdatedAt,
		)
		if err != nil {
			log.Printf("ListVideoByUsername: failed to scan video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
			return
		}
		videos = append(videos, video)
		if videoOwnerUID == "" {
			videoOwnerUID = video.UserUID
		}
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListVideoByUsername: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate videos")
		return
	}

	// Check if authenticated user follows the video owner
	following := false
	if auth && videoOwnerUID != "" {
		var hasFollow bool
		err := Mdb.DB.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM followers WHERE followed_by = $1 AND followed_to = $2)`,
			claims.UID, videoOwnerUID,
		).Scan(&hasFollow)
		if err == nil {
			following = hasFollow
		}
	}

	// Build response with following status for each video
	type VideoWithFollowing struct {
		Video     Videos `json:"video"`
		Following bool   `json:"following"`
	}

	videosWithFollowing := make([]VideoWithFollowing, len(videos))
	for i, video := range videos {
		videosWithFollowing[i] = VideoWithFollowing{
			Video:     video,
			Following: following,
		}
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"videos":   videosWithFollowing,
		"limit":    limit,
		"offset":   offset,
		"count":    len(videos),
		"username": username,
	})
}
