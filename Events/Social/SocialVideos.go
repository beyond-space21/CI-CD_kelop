package social

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	blake3 "lukechampine.com/blake3"

	Users "hifi/Events/Users"
	Videos "hifi/Events/Videos"
	Auth "hifi/Services/Auth"
	Mdb "hifi/Services/Mdb"
	Utils "hifi/Utils"
)

// nullStringToPtr converts sql.NullString to *string (nil if NULL, pointer to value if not)
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func HandleVideos(req chi.Router) {
	req.Post("/upvote/{videoID}", Upvote)
	req.Post("/downvote/{videoID}", Downvote)
	req.Post("/comment/{videoID}", Comment)
	req.Post("/reply/{commentID}", Reply)

	req.Get("/comments/{videoID}", ListComments)
	req.Get("/replies/{commentID}", ListReplies)
}

func Upvote(w http.ResponseWriter, r *http.Request) {
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

	// Check if video exists
	var video Videos.Videos
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
			log.Printf("Upvote: failed to fetch video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch video")
		}
		return
	}

	// Check if already upvoted
	var upvoteID int
	err = Mdb.DB.QueryRowContext(ctx,
		"SELECT id FROM upvotes WHERE upvoted_by = $1 AND upvoted_to = $2",
		claims.UID, videoID,
	).Scan(&upvoteID)
	if err == nil {
		// Already upvoted, remove downvote if exists
		result, delErr := Mdb.DB.ExecContext(ctx,
			"DELETE FROM downvotes WHERE downvoted_by = $1 AND downvoted_to = $2",
			claims.UID, videoID,
		)
		if delErr == nil {
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				_, updErr := Mdb.DB.ExecContext(ctx,
					"UPDATE videos SET video_downvotes = video_downvotes - 1 WHERE video_id = $1",
					videoID,
				)
				if updErr != nil {
					log.Printf("Upvote: failed to update video downvotes: %v", updErr)
					Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update video downvotes")
					return
				}
			}
		}
		Utils.SendSuccessResponse(w, map[string]string{"message": "Already upvoted, downvote removed if existed"})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("Upvote: failed to check existing upvote: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to check existing upvote")
		return
	}

	// Remove any existing downvote first
	result, delErr := Mdb.DB.ExecContext(ctx,
		"DELETE FROM downvotes WHERE downvoted_by = $1 AND downvoted_to = $2",
		claims.UID, videoID,
	)
	if delErr != nil {
		log.Printf("Upvote: failed to remove previous downvote: %v", delErr)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to remove previous downvote")
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		_, updErr := Mdb.DB.ExecContext(ctx,
			"UPDATE videos SET video_downvotes = video_downvotes - 1 WHERE video_id = $1",
			videoID,
		)
		if updErr != nil {
			log.Printf("Upvote: failed to update video downvotes: %v", updErr)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update video downvotes")
			return
		}
	}

	// Insert upvote (ignore if already exists due to UNIQUE constraint)
	_, err = Mdb.DB.ExecContext(ctx,
		"INSERT INTO upvotes (upvoted_by, upvoted_to, upvoted_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
		claims.UID, videoID, time.Now(),
	)
	if err != nil {
		log.Printf("Upvote: failed to upvote video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to upvote video")
		return
	}

	// Update video upvotes
	_, err = Mdb.DB.ExecContext(ctx,
		"UPDATE videos SET video_upvotes = video_upvotes + 1 WHERE video_id = $1",
		videoID,
	)
	if err != nil {
		log.Printf("Upvote: failed to update video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update video")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Video upvoted"})
}

func Downvote(w http.ResponseWriter, r *http.Request) {
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

	// Check if video exists
	var video Videos.Videos
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
			log.Printf("Downvote: failed to fetch video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch video")
		}
		return
	}

	// Check if already downvoted
	var downvoteID int
	err = Mdb.DB.QueryRowContext(ctx,
		"SELECT id FROM downvotes WHERE downvoted_by = $1 AND downvoted_to = $2",
		claims.UID, videoID,
	).Scan(&downvoteID)
	if err == nil {
		// Already downvoted, remove upvote if exists
		result, delErr := Mdb.DB.ExecContext(ctx,
			"DELETE FROM upvotes WHERE upvoted_by = $1 AND upvoted_to = $2",
			claims.UID, videoID,
		)
		if delErr == nil {
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				_, updErr := Mdb.DB.ExecContext(ctx,
					"UPDATE videos SET video_upvotes = video_upvotes - 1 WHERE video_id = $1",
					videoID,
				)
				if updErr != nil {
					log.Printf("Downvote: failed to update video upvotes: %v", updErr)
					Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update video upvotes")
					return
				}
			}
		}
		Utils.SendSuccessResponse(w, map[string]string{"message": "Already downvoted, upvote removed if existed"})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("Downvote: failed to check existing downvote: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to check existing downvote")
		return
	}

	// Remove any existing upvote first
	result, delErr := Mdb.DB.ExecContext(ctx,
		"DELETE FROM upvotes WHERE upvoted_by = $1 AND upvoted_to = $2",
		claims.UID, videoID,
	)
	if delErr != nil {
		log.Printf("Downvote: failed to remove previous upvote: %v", delErr)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to remove previous upvote")
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		_, updErr := Mdb.DB.ExecContext(ctx,
			"UPDATE videos SET video_upvotes = video_upvotes - 1 WHERE video_id = $1",
			videoID,
		)
		if updErr != nil {
			log.Printf("Downvote: failed to update video upvotes: %v", updErr)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update video upvotes")
			return
		}
	}

	// Insert downvote
	_, err = Mdb.DB.ExecContext(ctx,
		"INSERT INTO downvotes (downvoted_by, downvoted_to, downvoted_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
		claims.UID, videoID, time.Now(),
	)
	if err != nil {
		log.Printf("Downvote: failed to downvote video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to downvote video")
		return
	}

	// Update video downvotes
	_, err = Mdb.DB.ExecContext(ctx,
		"UPDATE videos SET video_downvotes = video_downvotes + 1 WHERE video_id = $1",
		videoID,
	)
	if err != nil {
		log.Printf("Downvote: failed to update video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update video")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Video downvoted"})
}

func Comment(w http.ResponseWriter, r *http.Request) {
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Comment: failed to read body: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to read body")
		return
	}

	var comment map[string]interface{}
	err = json.Unmarshal(body, &comment)
	if err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Failed to unmarshal body")
		return
	}

	// Check if video exists
	var video Videos.Videos
	err = Mdb.DB.QueryRowContext(ctx,
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
			log.Printf("Comment: failed to fetch video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch video")
		}
		return
	}

	// Get user
	var user Users.User
	var bioNull, emailNull sql.NullString
	err = Mdb.DB.QueryRowContext(ctx,
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
			log.Printf("Comment: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return
	}
	user.Bio = nullStringToPtr(bioNull)
	user.Email = nullStringToPtr(emailNull)

	// Create comment
	commentID := fmt.Sprintf("%x", blake3.Sum256([]byte(claims.UID+time.Now().Format(time.RFC3339)+uuid.New().String()+videoID)))
	commentText, ok := comment["comment"].(string)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Comment text is required")
		return
	}

	_, err = Mdb.DB.ExecContext(ctx,
		`INSERT INTO comments (comment_id, commented_by, commented_to, commented_at, comment, comment_by_username, total_replies)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		commentID, claims.UID, videoID, time.Now(), commentText, user.Username, 0,
	)
	if err != nil {
		log.Printf("Comment: failed to comment on video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to comment on video")
		return
	}

	// Update video comment count
	_, err = Mdb.DB.ExecContext(ctx,
		"UPDATE videos SET video_comments = video_comments + 1 WHERE video_id = $1",
		videoID,
	)
	if err != nil {
		log.Printf("Comment: failed to update video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update video")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Video commented"})
}

func Reply(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	commentID := chi.URLParam(r, "commentID")
	if commentID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Comment ID is required")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Reply: failed to read body: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to read body")
		return
	}

	var reply map[string]interface{}
	err = json.Unmarshal(body, &reply)
	if err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Failed to unmarshal body")
		return
	}

	// Check if comment exists
	var comment Comments
	err = Mdb.DB.QueryRowContext(ctx,
		`SELECT id, comment_id, commented_by, commented_to, commented_at, comment, 
			comment_by_username, total_replies
		FROM comments WHERE comment_id = $1`,
		commentID,
	).Scan(
		&comment.ID, &comment.CommentID, &comment.CommentedBy, &comment.CommentedTo,
		&comment.CommentedAt, &comment.Comment, &comment.CommentByUsername,
		&comment.TotalReplies,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "Comment not found")
		} else {
			log.Printf("Reply: failed to fetch comment: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch comment")
		}
		return
	}

	// Get user
	var user Users.User
	var bioNull, emailNull sql.NullString
	err = Mdb.DB.QueryRowContext(ctx,
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
			log.Printf("Reply: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return
	}

	replyText, ok := reply["reply"].(string)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Reply text is required")
		return
	}

	// Check if reply already exists
	var existingReplyID int
	err = Mdb.DB.QueryRowContext(ctx,
		"SELECT id FROM replies WHERE replied_to = $1 AND replied_by = $2",
		commentID, claims.UID,
	).Scan(&existingReplyID)
	if err == nil {
		// Reply exists, update it
		_, err = Mdb.DB.ExecContext(ctx,
			"UPDATE replies SET reply = $1 WHERE replied_to = $2 AND replied_by = $3",
			replyText, commentID, claims.UID,
		)
		if err != nil {
			log.Printf("Reply: failed to update reply: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update reply to comment")
			return
		}
		Utils.SendSuccessResponse(w, map[string]string{"message": "Reply updated"})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("Reply: failed to check existing reply: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to check existing reply")
		return
	}

	// Insert new reply
	replyID := fmt.Sprintf("%x", blake3.Sum256([]byte(claims.UID+time.Now().Format(time.RFC3339)+uuid.New().String()+commentID)))
	_, err = Mdb.DB.ExecContext(ctx,
		`INSERT INTO replies (reply_id, replied_by, replied_to, replied_at, reply, reply_by_username)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		replyID, claims.UID, commentID, time.Now(), replyText, user.Username,
	)
	if err != nil {
		log.Printf("Reply: failed to insert reply: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to insert reply")
		return
	}

	// Update comment reply count
	_, err = Mdb.DB.ExecContext(ctx,
		"UPDATE comments SET total_replies = total_replies + 1 WHERE comment_id = $1",
		commentID,
	)
	if err != nil {
		log.Printf("Reply: failed to update comment: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update comment")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Reply added"})
}

func ListComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	videoID := chi.URLParam(r, "videoID")
	if videoID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Video ID is required")
		return
	}

	// Get pagination parameters from query string
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get comments ordered by timestamp (newest first) with total count in single query
	// Use window function COUNT(*) OVER() to get total count without separate query
	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT id, comment_id, commented_by, commented_to, commented_at, comment, 
			comment_by_username, total_replies,
			COUNT(*) OVER() as total_count
		FROM comments 
		WHERE commented_to = $1 
		ORDER BY commented_at DESC
		LIMIT $2 OFFSET $3`,
		videoID, limit, offset,
	)
	if err != nil {
		log.Printf("ListComments: failed to find comments: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to find comments")
		return
	}
	defer rows.Close()

	var comments []Comments
	var count int
	for rows.Next() {
		var comment Comments
		err := rows.Scan(
			&comment.ID, &comment.CommentID, &comment.CommentedBy, &comment.CommentedTo,
			&comment.CommentedAt, &comment.Comment, &comment.CommentByUsername,
			&comment.TotalReplies, &count, // total_count from window function
		)
		if err != nil {
			log.Printf("ListComments: failed to scan comment: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode comment")
			return
		}
		comments = append(comments, comment)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListComments: failed to iterate comments: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate comments")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"comments": comments,
		"limit":    limit,
		"offset":   offset,
		"count":    count,
	})
}

func ListReplies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID := chi.URLParam(r, "commentID")
	if commentID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Comment ID is required")
		return
	}

	// Get pagination parameters from query string
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get replies ordered by timestamp (newest first) with total count in single query
	// Use window function COUNT(*) OVER() to get total count without separate query
	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT id, reply_id, replied_by, replied_to, replied_at, reply, reply_by_username,
			COUNT(*) OVER() as total_count
		FROM replies 
		WHERE replied_to = $1 
		ORDER BY replied_at DESC
		LIMIT $2 OFFSET $3`,
		commentID, limit, offset,
	)
	if err != nil {
		log.Printf("ListReplies: failed to find replies: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to find replies")
		return
	}
	defer rows.Close()

	var replies []Replies
	var count int
	for rows.Next() {
		var reply Replies
		err := rows.Scan(
			&reply.ID, &reply.ReplyID, &reply.RepliedBy, &reply.RepliedTo,
			&reply.RepliedAt, &reply.Reply, &reply.ReplyByUsername, &count, // total_count from window function
		)
		if err != nil {
			log.Printf("ListReplies: failed to scan reply: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode reply")
			return
		}
		replies = append(replies, reply)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListReplies: failed to iterate replies: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate replies")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"replies": replies,
		"limit":   limit,
		"offset":  offset,
		"count":   count,
	})
}

func View(ctx context.Context, auth bool, claims *Auth.Token, videoID string) error {
	// Increment video views counter (simple count, no authentication-based tracking)
	_, err := Mdb.DB.ExecContext(ctx,
		"UPDATE videos SET video_views = video_views + 1 WHERE video_id = $1",
		videoID,
	)
	if err != nil {
		return err
	}

	return nil
}
