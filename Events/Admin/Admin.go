package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	Search "hifi/Events/Search"
	Social "hifi/Events/Social"
	Users "hifi/Events/Users"
	Videos "hifi/Events/Videos"
	Auth "hifi/Services/Auth"
	Mdb "hifi/Services/Mdb"
	storage "hifi/Services/Storage"
	Utils "hifi/Utils"
)

var GetClaims func(r *http.Request) (*Auth.Token, bool) = Auth.GetClaims

// Handle sets up the routes for admin endpoints
func Handle(r chi.Router) {
	// List endpoints
	r.Get("/users", ListUsers)
	r.Get("/videos", ListVideos)
	r.Get("/comments", ListComments)
	r.Get("/replies", ListReplies)
	r.Get("/followers", ListFollowers)
	r.Get("/counters", GetCounters)
	r.Post("/counters/resync", ResyncCounters)

	// Delete endpoints
	r.Delete("/users/{uid}", DeleteUser)
	r.Delete("/videos/{videoID}", DeleteVideo)
	r.Delete("/comments/{commentID}", DeleteComment)
	r.Delete("/replies/{replyID}", DeleteReply)
}

// requireAdmin checks if the authenticated user has admin role
func requireAdmin(w http.ResponseWriter, r *http.Request) (*Users.User, bool) {
	ctx := r.Context()
	claims, ok := GetClaims(r)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}

	// Fetch user to check role
	user, err := fetchUserByUID(ctx, claims.UID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("requireAdmin: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return nil, false
	}

	if user.Role != "admin" {
		Utils.SendErrorResponse(w, http.StatusForbidden, "Forbidden: admin access required")
		return nil, false
	}

	return user, true
}

// fetchUserByUID retrieves a user by their UID
func fetchUserByUID(ctx context.Context, uid string) (*Users.User, error) {
	var user Users.User
	var bioNull, emailNull sql.NullString
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT id, uid, username, name, role, profile_picture, bio, email, 
			followers, following, total_streams, total_videos, created_at, updated_at
		FROM users WHERE uid = $1`,
		uid,
	).Scan(
		&user.ID, &user.UID, &user.Username, &user.Name, &user.Role,
		&user.ProfilePicture, &bioNull, &emailNull, &user.Followers, &user.Following,
		&user.TotalStreams, &user.TotalVideos, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("fetchUserByUID: %w", err)
	}
	user.Bio = nullStringToPtr(bioNull)
	user.Email = nullStringToPtr(emailNull)
	return &user, nil
}

// nullStringToPtr converts sql.NullString to *string (nil if NULL, pointer to value if not)
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// getStringValue converts *string to interface{} for SQL (nil becomes NULL)
func getStringValue(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// ListUsers lists all users with pagination and optional filters
func ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse filter parameters
	usernameFilter := strings.TrimSpace(r.URL.Query().Get("username"))
	nameFilter := strings.TrimSpace(r.URL.Query().Get("name"))
	roleFilter := strings.TrimSpace(r.URL.Query().Get("role"))
	uidFilter := strings.TrimSpace(r.URL.Query().Get("uid"))

	// Numeric range filters
	followersMinStr := r.URL.Query().Get("followers_min")
	followersMaxStr := r.URL.Query().Get("followers_max")
	followingMinStr := r.URL.Query().Get("following_min")
	followingMaxStr := r.URL.Query().Get("following_max")
	totalVideosMinStr := r.URL.Query().Get("total_videos_min")
	totalVideosMaxStr := r.URL.Query().Get("total_videos_max")

	// Date range filters
	createdAfterStr := r.URL.Query().Get("created_after")
	createdBeforeStr := r.URL.Query().Get("created_before")
	updatedAfterStr := r.URL.Query().Get("updated_after")
	updatedBeforeStr := r.URL.Query().Get("updated_before")

	// Build query with filters
	query := `SELECT id, uid, username, name, role, profile_picture, bio, email, 
		followers, following, total_streams, total_videos, created_at, updated_at
		FROM users`
	args := []interface{}{}
	argPos := 1
	conditions := []string{}

	// Username filter (case-insensitive partial match)
	if usernameFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(username) LIKE $%d", argPos))
		args = append(args, "%"+strings.ToLower(usernameFilter)+"%")
		argPos++
	}

	// Name filter (case-insensitive partial match)
	if nameFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE $%d", argPos))
		args = append(args, "%"+strings.ToLower(nameFilter)+"%")
		argPos++
	}

	// Role filter (exact match, case-insensitive)
	if roleFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(role) = $%d", argPos))
		args = append(args, strings.ToLower(roleFilter))
		argPos++
	}

	// UID filter (exact match)
	if uidFilter != "" {
		conditions = append(conditions, fmt.Sprintf("uid = $%d", argPos))
		args = append(args, uidFilter)
		argPos++
	}

	// Followers range filters
	if followersMinStr != "" {
		if min, err := strconv.Atoi(followersMinStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("followers >= $%d", argPos))
			args = append(args, min)
			argPos++
		}
	}
	if followersMaxStr != "" {
		if max, err := strconv.Atoi(followersMaxStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("followers <= $%d", argPos))
			args = append(args, max)
			argPos++
		}
	}

	// Following range filters
	if followingMinStr != "" {
		if min, err := strconv.Atoi(followingMinStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("following >= $%d", argPos))
			args = append(args, min)
			argPos++
		}
	}
	if followingMaxStr != "" {
		if max, err := strconv.Atoi(followingMaxStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("following <= $%d", argPos))
			args = append(args, max)
			argPos++
		}
	}

	// Total videos range filters
	if totalVideosMinStr != "" {
		if min, err := strconv.Atoi(totalVideosMinStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("total_videos >= $%d", argPos))
			args = append(args, min)
			argPos++
		}
	}
	if totalVideosMaxStr != "" {
		if max, err := strconv.Atoi(totalVideosMaxStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("total_videos <= $%d", argPos))
			args = append(args, max)
			argPos++
		}
	}

	// Created date range filters
	if createdAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAfterStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}
	if createdBeforeStr != "" {
		if t, err := time.Parse(time.RFC3339, createdBeforeStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}

	// Updated date range filters
	if updatedAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedAfterStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("updated_at >= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}
	if updatedBeforeStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedBeforeStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("updated_at <= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}

	// Add WHERE clause if there are conditions
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argPos) + " OFFSET $" + strconv.Itoa(argPos+1)
	args = append(args, limit, offset)

	rows, err := Mdb.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("ListUsers: failed to query users: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer rows.Close()

	var users []Users.User
	for rows.Next() {
		var user Users.User
		var bioNull, emailNull sql.NullString
		if err := rows.Scan(
			&user.ID, &user.UID, &user.Username, &user.Name, &user.Role,
			&user.ProfilePicture, &bioNull, &emailNull, &user.Followers, &user.Following,
			&user.TotalStreams, &user.TotalVideos, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			log.Printf("ListUsers: failed to scan user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch users")
			return
		}
		user.Bio = nullStringToPtr(bioNull)
		user.Email = nullStringToPtr(emailNull)
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		log.Printf("ListUsers: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate users")
		return
	}

	// Build filters map for response
	filters := make(map[string]interface{})
	if usernameFilter != "" {
		filters["username"] = usernameFilter
	}
	if nameFilter != "" {
		filters["name"] = nameFilter
	}
	if roleFilter != "" {
		filters["role"] = roleFilter
	}
	if uidFilter != "" {
		filters["uid"] = uidFilter
	}
	if followersMinStr != "" || followersMaxStr != "" {
		filters["followers"] = map[string]string{
			"min": followersMinStr,
			"max": followersMaxStr,
		}
	}
	if followingMinStr != "" || followingMaxStr != "" {
		filters["following"] = map[string]string{
			"min": followingMinStr,
			"max": followingMaxStr,
		}
	}
	if totalVideosMinStr != "" || totalVideosMaxStr != "" {
		filters["total_videos"] = map[string]string{
			"min": totalVideosMinStr,
			"max": totalVideosMaxStr,
		}
	}
	if createdAfterStr != "" || createdBeforeStr != "" {
		filters["created_at"] = map[string]string{
			"after":  createdAfterStr,
			"before": createdBeforeStr,
		}
	}
	if updatedAfterStr != "" || updatedBeforeStr != "" {
		filters["updated_at"] = map[string]string{
			"after":  updatedAfterStr,
			"before": updatedBeforeStr,
		}
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"users":   users,
		"limit":   limit,
		"offset":  offset,
		"count":   len(users),
		"filters": filters,
	})
}

// ListVideos lists all videos with pagination and optional filters
func ListVideos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse filter parameters
	videoIDFilter := strings.TrimSpace(r.URL.Query().Get("video_id"))
	videoTitleFilter := strings.TrimSpace(r.URL.Query().Get("video_title"))
	videoDescriptionFilter := strings.TrimSpace(r.URL.Query().Get("video_description"))
	userUsernameFilter := strings.TrimSpace(r.URL.Query().Get("user_username"))
	userUIDFilter := strings.TrimSpace(r.URL.Query().Get("user_uid"))
	videoTagFilter := strings.TrimSpace(r.URL.Query().Get("video_tag"))

	// Numeric range filters
	videoViewsMinStr := r.URL.Query().Get("video_views_min")
	videoViewsMaxStr := r.URL.Query().Get("video_views_max")
	videoUpvotesMinStr := r.URL.Query().Get("video_upvotes_min")
	videoUpvotesMaxStr := r.URL.Query().Get("video_upvotes_max")
	videoDownvotesMinStr := r.URL.Query().Get("video_downvotes_min")
	videoDownvotesMaxStr := r.URL.Query().Get("video_downvotes_max")
	videoCommentsMinStr := r.URL.Query().Get("video_comments_min")
	videoCommentsMaxStr := r.URL.Query().Get("video_comments_max")

	// Date range filters
	createdAfterStr := r.URL.Query().Get("created_after")
	createdBeforeStr := r.URL.Query().Get("created_before")
	updatedAfterStr := r.URL.Query().Get("updated_after")
	updatedBeforeStr := r.URL.Query().Get("updated_before")

	// Build query with filters
	query := `SELECT id, video_id, video_url, video_thumbnail, video_title, video_description, 
		video_tags, video_views, video_upvotes, video_downvotes, video_comments, 
		user_uid, user_username, created_at, updated_at
		FROM videos`
	args := []interface{}{}
	argPos := 1
	conditions := []string{}

	// Video ID filter (exact match)
	if videoIDFilter != "" {
		conditions = append(conditions, fmt.Sprintf("video_id = $%d", argPos))
		args = append(args, videoIDFilter)
		argPos++
	}

	// Video title filter (case-insensitive partial match)
	if videoTitleFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(video_title) LIKE $%d", argPos))
		args = append(args, "%"+strings.ToLower(videoTitleFilter)+"%")
		argPos++
	}

	// Video description filter (case-insensitive partial match)
	if videoDescriptionFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(video_description) LIKE $%d", argPos))
		args = append(args, "%"+strings.ToLower(videoDescriptionFilter)+"%")
		argPos++
	}

	// User username filter (case-insensitive partial match)
	if userUsernameFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(user_username) LIKE $%d", argPos))
		args = append(args, "%"+strings.ToLower(userUsernameFilter)+"%")
		argPos++
	}

	// User UID filter (exact match)
	if userUIDFilter != "" {
		conditions = append(conditions, fmt.Sprintf("user_uid = $%d", argPos))
		args = append(args, userUIDFilter)
		argPos++
	}

	// Video tag filter (case-insensitive, searches within video_tags array)
	if videoTagFilter != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM unnest(video_tags) AS tag WHERE LOWER(tag) LIKE $%d)", argPos))
		args = append(args, "%"+strings.ToLower(videoTagFilter)+"%")
		argPos++
	}

	// Video views range filters
	if videoViewsMinStr != "" {
		if min, err := strconv.Atoi(videoViewsMinStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_views >= $%d", argPos))
			args = append(args, min)
			argPos++
		}
	}
	if videoViewsMaxStr != "" {
		if max, err := strconv.Atoi(videoViewsMaxStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_views <= $%d", argPos))
			args = append(args, max)
			argPos++
		}
	}

	// Video upvotes range filters
	if videoUpvotesMinStr != "" {
		if min, err := strconv.Atoi(videoUpvotesMinStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_upvotes >= $%d", argPos))
			args = append(args, min)
			argPos++
		}
	}
	if videoUpvotesMaxStr != "" {
		if max, err := strconv.Atoi(videoUpvotesMaxStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_upvotes <= $%d", argPos))
			args = append(args, max)
			argPos++
		}
	}

	// Video downvotes range filters
	if videoDownvotesMinStr != "" {
		if min, err := strconv.Atoi(videoDownvotesMinStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_downvotes >= $%d", argPos))
			args = append(args, min)
			argPos++
		}
	}
	if videoDownvotesMaxStr != "" {
		if max, err := strconv.Atoi(videoDownvotesMaxStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_downvotes <= $%d", argPos))
			args = append(args, max)
			argPos++
		}
	}

	// Video comments range filters
	if videoCommentsMinStr != "" {
		if min, err := strconv.Atoi(videoCommentsMinStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_comments >= $%d", argPos))
			args = append(args, min)
			argPos++
		}
	}
	if videoCommentsMaxStr != "" {
		if max, err := strconv.Atoi(videoCommentsMaxStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("video_comments <= $%d", argPos))
			args = append(args, max)
			argPos++
		}
	}

	// Created date range filters
	if createdAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAfterStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}
	if createdBeforeStr != "" {
		if t, err := time.Parse(time.RFC3339, createdBeforeStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}

	// Updated date range filters
	if updatedAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedAfterStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("updated_at >= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}
	if updatedBeforeStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedBeforeStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("updated_at <= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}

	// Add WHERE clause if there are conditions
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argPos) + " OFFSET $" + strconv.Itoa(argPos+1)
	args = append(args, limit, offset)

	rows, err := Mdb.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("ListVideos: failed to query videos: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
		return
	}
	defer rows.Close()

	var videos []Videos.Videos
	for rows.Next() {
		var video Videos.Videos
		if err := rows.Scan(
			&video.ID, &video.VideoID, &video.VideoURL, &video.VideoThumbnail,
			&video.VideoTitle, &video.VideoDescription, &video.VideoTags,
			&video.VideoViews, &video.VideoUpvotes, &video.VideoDownvotes,
			&video.VideoComments, &video.UserUID, &video.UserUsername,
			&video.CreatedAt, &video.UpdatedAt,
		); err != nil {
			log.Printf("ListVideos: failed to scan video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch videos")
			return
		}
		videos = append(videos, video)
	}

	if err := rows.Err(); err != nil {
		log.Printf("ListVideos: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate videos")
		return
	}

	// Build filters map for response
	filters := make(map[string]interface{})
	if videoIDFilter != "" {
		filters["video_id"] = videoIDFilter
	}
	if videoTitleFilter != "" {
		filters["video_title"] = videoTitleFilter
	}
	if videoDescriptionFilter != "" {
		filters["video_description"] = videoDescriptionFilter
	}
	if userUsernameFilter != "" {
		filters["user_username"] = userUsernameFilter
	}
	if userUIDFilter != "" {
		filters["user_uid"] = userUIDFilter
	}
	if videoTagFilter != "" {
		filters["video_tag"] = videoTagFilter
	}
	if videoViewsMinStr != "" || videoViewsMaxStr != "" {
		filters["video_views"] = map[string]string{
			"min": videoViewsMinStr,
			"max": videoViewsMaxStr,
		}
	}
	if videoUpvotesMinStr != "" || videoUpvotesMaxStr != "" {
		filters["video_upvotes"] = map[string]string{
			"min": videoUpvotesMinStr,
			"max": videoUpvotesMaxStr,
		}
	}
	if videoDownvotesMinStr != "" || videoDownvotesMaxStr != "" {
		filters["video_downvotes"] = map[string]string{
			"min": videoDownvotesMinStr,
			"max": videoDownvotesMaxStr,
		}
	}
	if videoCommentsMinStr != "" || videoCommentsMaxStr != "" {
		filters["video_comments"] = map[string]string{
			"min": videoCommentsMinStr,
			"max": videoCommentsMaxStr,
		}
	}
	if createdAfterStr != "" || createdBeforeStr != "" {
		filters["created_at"] = map[string]string{
			"after":  createdAfterStr,
			"before": createdBeforeStr,
		}
	}
	if updatedAfterStr != "" || updatedBeforeStr != "" {
		filters["updated_at"] = map[string]string{
			"after":  updatedAfterStr,
			"before": updatedBeforeStr,
		}
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"videos":  videos,
		"limit":   limit,
		"offset":  offset,
		"count":   len(videos),
		"filters": filters,
	})
}

// ListComments lists all comments with pagination and optional filters
func ListComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	filterStr := r.URL.Query().Get("filter") // Filter by comment_id, comment text, username, video_id, etc.

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build query with optional filter
	query := `SELECT id, comment_id, commented_by, commented_to, commented_at, comment, 
		comment_by_username, total_replies
		FROM comments`
	args := []interface{}{}
	argPos := 1

	if filterStr != "" {
		filterStr = strings.ToLower(strings.TrimSpace(filterStr))
		query += fmt.Sprintf(" WHERE LOWER(comment_id) LIKE $%d OR LOWER(comment) LIKE $%d OR LOWER(comment_by_username) LIKE $%d OR LOWER(commented_to) LIKE $%d",
			argPos, argPos, argPos, argPos)
		args = append(args, "%"+filterStr+"%")
		argPos++
	}

	query += " ORDER BY commented_at DESC LIMIT $" + strconv.Itoa(argPos) + " OFFSET $" + strconv.Itoa(argPos+1)
	args = append(args, limit, offset)

	rows, err := Mdb.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("ListComments: failed to query comments: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch comments")
		return
	}
	defer rows.Close()

	var comments []Social.Comments
	for rows.Next() {
		var comment Social.Comments
		if err := rows.Scan(
			&comment.ID, &comment.CommentID, &comment.CommentedBy, &comment.CommentedTo,
			&comment.CommentedAt, &comment.Comment, &comment.CommentByUsername,
			&comment.TotalReplies,
		); err != nil {
			log.Printf("ListComments: failed to scan comment: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch comments")
			return
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		log.Printf("ListComments: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate comments")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"comments": comments,
		"limit":    limit,
		"offset":   offset,
		"count":    len(comments),
		"filter":   filterStr,
	})
}

// ListReplies lists all replies with pagination and optional filters
func ListReplies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	filterStr := r.URL.Query().Get("filter") // Filter by reply_id, reply text, username, comment_id, etc.

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build query with optional filter
	query := `SELECT id, reply_id, replied_by, replied_to, replied_at, reply, reply_by_username
		FROM replies`
	args := []interface{}{}
	argPos := 1

	if filterStr != "" {
		filterStr = strings.ToLower(strings.TrimSpace(filterStr))
		query += fmt.Sprintf(" WHERE LOWER(reply_id) LIKE $%d OR LOWER(reply) LIKE $%d OR LOWER(reply_by_username) LIKE $%d OR LOWER(replied_to) LIKE $%d",
			argPos, argPos, argPos, argPos)
		args = append(args, "%"+filterStr+"%")
		argPos++
	}

	query += " ORDER BY replied_at DESC LIMIT $" + strconv.Itoa(argPos) + " OFFSET $" + strconv.Itoa(argPos+1)
	args = append(args, limit, offset)

	rows, err := Mdb.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("ListReplies: failed to query replies: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch replies")
		return
	}
	defer rows.Close()

	var replies []Social.Replies
	for rows.Next() {
		var reply Social.Replies
		if err := rows.Scan(
			&reply.ID, &reply.ReplyID, &reply.RepliedBy, &reply.RepliedTo,
			&reply.RepliedAt, &reply.Reply, &reply.ReplyByUsername,
		); err != nil {
			log.Printf("ListReplies: failed to scan reply: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch replies")
			return
		}
		replies = append(replies, reply)
	}

	if err := rows.Err(); err != nil {
		log.Printf("ListReplies: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate replies")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"replies": replies,
		"limit":   limit,
		"offset":  offset,
		"count":   len(replies),
		"filter":  filterStr,
	})
}

// ListFollowers lists all follower relationships with pagination and optional filters
func ListFollowers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse filter parameters
	followedByUsernameFilter := strings.TrimSpace(r.URL.Query().Get("followed_by_username"))
	followedToUsernameFilter := strings.TrimSpace(r.URL.Query().Get("followed_to_username"))
	followedByUIDFilter := strings.TrimSpace(r.URL.Query().Get("followed_by"))
	followedToUIDFilter := strings.TrimSpace(r.URL.Query().Get("followed_to"))

	// Date range filters
	followedAfterStr := r.URL.Query().Get("followed_after")
	followedBeforeStr := r.URL.Query().Get("followed_before")

	// Build query with filters
	query := `SELECT f.id, f.followed_by, u1.username as followed_by_username, 
		f.followed_to, u2.username as followed_to_username, f.followed_at
		FROM followers f
		JOIN users u1 ON f.followed_by = u1.uid
		JOIN users u2 ON f.followed_to = u2.uid`
	args := []interface{}{}
	argPos := 1
	conditions := []string{}

	// Followed by username filter (case-insensitive partial match)
	if followedByUsernameFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(u1.username) LIKE $%d", argPos))
		args = append(args, "%"+strings.ToLower(followedByUsernameFilter)+"%")
		argPos++
	}

	// Followed to username filter (case-insensitive partial match)
	if followedToUsernameFilter != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(u2.username) LIKE $%d", argPos))
		args = append(args, "%"+strings.ToLower(followedToUsernameFilter)+"%")
		argPos++
	}

	// Followed by UID filter (exact match)
	if followedByUIDFilter != "" {
		conditions = append(conditions, fmt.Sprintf("f.followed_by = $%d", argPos))
		args = append(args, followedByUIDFilter)
		argPos++
	}

	// Followed to UID filter (exact match)
	if followedToUIDFilter != "" {
		conditions = append(conditions, fmt.Sprintf("f.followed_to = $%d", argPos))
		args = append(args, followedToUIDFilter)
		argPos++
	}

	// Followed date range filters
	if followedAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, followedAfterStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("f.followed_at >= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}
	if followedBeforeStr != "" {
		if t, err := time.Parse(time.RFC3339, followedBeforeStr); err == nil {
			conditions = append(conditions, fmt.Sprintf("f.followed_at <= $%d", argPos))
			args = append(args, t)
			argPos++
		}
	}

	// Add WHERE clause if there are conditions
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY f.followed_at DESC LIMIT $" + strconv.Itoa(argPos) + " OFFSET $" + strconv.Itoa(argPos+1)
	args = append(args, limit, offset)

	rows, err := Mdb.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("ListFollowers: failed to query followers: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch followers")
		return
	}
	defer rows.Close()

	var followers []Social.Followers
	for rows.Next() {
		var follower Social.Followers
		if err := rows.Scan(
			&follower.ID, &follower.FollowedBy, &follower.FollowedByUsername,
			&follower.FollowedTo, &follower.FollowedToUsername, &follower.FollowedAt,
		); err != nil {
			log.Printf("ListFollowers: failed to scan follower: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch followers")
			return
		}
		followers = append(followers, follower)
	}

	if err := rows.Err(); err != nil {
		log.Printf("ListFollowers: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate followers")
		return
	}

	// Build filters map for response
	filters := make(map[string]interface{})
	if followedByUsernameFilter != "" {
		filters["followed_by_username"] = followedByUsernameFilter
	}
	if followedToUsernameFilter != "" {
		filters["followed_to_username"] = followedToUsernameFilter
	}
	if followedByUIDFilter != "" {
		filters["followed_by"] = followedByUIDFilter
	}
	if followedToUIDFilter != "" {
		filters["followed_to"] = followedToUIDFilter
	}
	if followedAfterStr != "" || followedBeforeStr != "" {
		filters["followed_at"] = map[string]string{
			"after":  followedAfterStr,
			"before": followedBeforeStr,
		}
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"followers": followers,
		"limit":     limit,
		"offset":    offset,
		"count":     len(followers),
		"filters":   filters,
	})
}

// DeleteUser deletes a user by UID (admin only)
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	uid := chi.URLParam(r, "uid")
	if uid == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "User UID is required")
		return
	}

	// Check if user exists
	var existing Users.User
	var bioNull, emailNull sql.NullString
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT id, uid, username, name, role, profile_picture, bio, email, 
			followers, following, total_streams, total_videos, created_at, updated_at
		FROM users WHERE uid = $1`,
		uid,
	).Scan(
		&existing.ID, &existing.UID, &existing.Username, &existing.Name, &existing.Role,
		&existing.ProfilePicture, &bioNull, &emailNull, &existing.Followers, &existing.Following,
		&existing.TotalStreams, &existing.TotalVideos, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("DeleteUser: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load user")
		}
		return
	}
	existing.Bio = nullStringToPtr(bioNull)
	existing.Email = nullStringToPtr(emailNull)

	// Use transaction for atomicity
	tx, err := Mdb.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("DeleteUser: failed to begin transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Insert into deleted_users
	_, err = tx.ExecContext(ctx,
		`INSERT INTO deleted_users (uid, username, name, role, profile_picture, bio, email, followers, following, 
			total_streams, total_videos, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		existing.UID, existing.Username, existing.Name, existing.Role, existing.ProfilePicture,
		existing.Bio, existing.Email, existing.Followers, existing.Following, existing.TotalStreams, existing.TotalVideos,
		existing.CreatedAt, existing.UpdatedAt, time.Now(),
	)
	if err != nil {
		log.Printf("DeleteUser: failed to insert deleted user: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to archive deleted user")
		return
	}

	// Delete from users (CASCADE will handle related data)
	_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE uid = $1", existing.UID)
	if err != nil {
		log.Printf("DeleteUser: failed to delete user: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("DeleteUser: failed to commit transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to complete deletion")
		return
	}

	// Delete user from Elasticsearch (non-blocking, log errors but don't fail deletion)
	go func() {
		esCtx := context.Background()
		if err := Search.DeleteUser(esCtx, existing.UID); err != nil {
			log.Printf("DeleteUser: failed to delete user from Elasticsearch: %v", err)
		}
	}()

	Utils.SendSuccessResponse(w, map[string]string{"message": "User deleted successfully"})
}

// DeleteVideo deletes a video by videoID (admin only)
func DeleteVideo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
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
			log.Printf("DeleteVideo: failed to fetch video: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load video")
		}
		return
	}

	// Delete video and thumbnail files from R2 storage
	// Use the stored paths from the database
	video_obj_key := video.VideoURL
	thumbnail_obj_key := video.VideoThumbnail

	log.Printf("DeleteVideo: Video URL from DB: %s", video_obj_key)
	log.Printf("DeleteVideo: Thumbnail URL from DB: %s", thumbnail_obj_key)

	// Verify files exist before attempting deletion
	videoExists, _ := storage.IsFileExists(video_obj_key)
	thumbnailExists, _ := storage.IsFileExists(thumbnail_obj_key)
	log.Printf("DeleteVideo: Video exists in R2: %v, Thumbnail exists in R2: %v", videoExists, thumbnailExists)

	// Delete video file from storage
	if videoExists {
		if err := storage.DeleteFile(ctx, video_obj_key); err != nil {
			log.Printf("DeleteVideo: CRITICAL ERROR - failed to delete video file from storage (%s): %v", video_obj_key, err)
			// Continue with deletion even if storage cleanup fails
		} else {
			// Verify deletion by checking if file still exists
			exists, checkErr := storage.IsFileExists(video_obj_key)
			if checkErr == nil && exists {
				log.Printf("DeleteVideo: CRITICAL WARNING - video file still exists after deletion attempt: %s", video_obj_key)
			} else {
				log.Printf("DeleteVideo: successfully deleted and verified video file: %s", video_obj_key)
			}
		}
	} else {
		log.Printf("DeleteVideo: video file does not exist in R2, skipping deletion: %s", video_obj_key)
	}

	// Delete thumbnail file from storage
	if thumbnailExists {
		if err := storage.DeleteFile(ctx, thumbnail_obj_key); err != nil {
			log.Printf("DeleteVideo: CRITICAL ERROR - failed to delete thumbnail file from storage (%s): %v", thumbnail_obj_key, err)
			// Continue with deletion even if storage cleanup fails
		} else {
			// Verify deletion by checking if file still exists
			exists, checkErr := storage.IsFileExists(thumbnail_obj_key)
			if checkErr == nil && exists {
				log.Printf("DeleteVideo: CRITICAL WARNING - thumbnail file still exists after deletion attempt: %s", thumbnail_obj_key)
			} else {
				log.Printf("DeleteVideo: successfully deleted and verified thumbnail file: %s", thumbnail_obj_key)
			}
		}
	} else {
		log.Printf("DeleteVideo: thumbnail file does not exist in R2, skipping deletion: %s", thumbnail_obj_key)
	}

	// Use transaction for atomicity
	tx, err := Mdb.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("DeleteVideo: failed to begin transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Delete video (CASCADE will handle related data: upvotes, downvotes, comments, replies, views)
	_, err = tx.ExecContext(ctx, "DELETE FROM videos WHERE video_id = $1", videoID)
	if err != nil {
		log.Printf("DeleteVideo: failed to delete video: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete video")
		return
	}

	// Update user's total_videos count
	_, err = tx.ExecContext(ctx,
		"UPDATE users SET total_videos = total_videos - 1 WHERE uid = $1",
		video.UserUID,
	)
	if err != nil {
		log.Printf("DeleteVideo: failed to update user video count: %v", err)
		// Don't fail the request, just log the error
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("DeleteVideo: failed to commit transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to complete deletion")
		return
	}

	// Delete video from Elasticsearch (non-blocking, log errors but don't fail deletion)
	go func() {
		esCtx := context.Background()
		if err := Search.DeleteVideo(esCtx, videoID); err != nil {
			log.Printf("DeleteVideo: failed to delete video from Elasticsearch: %v", err)
		}
	}()

	Utils.SendSuccessResponse(w, map[string]string{"message": "Video deleted successfully"})
}

// DeleteComment deletes a comment by commentID (admin only)
func DeleteComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	commentID := chi.URLParam(r, "commentID")
	if commentID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Comment ID is required")
		return
	}

	// Check if comment exists and get video_id
	var commentedTo string
	err := Mdb.DB.QueryRowContext(ctx,
		"SELECT commented_to FROM comments WHERE comment_id = $1",
		commentID,
	).Scan(&commentedTo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "Comment not found")
		} else {
			log.Printf("DeleteComment: failed to fetch comment: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load comment")
		}
		return
	}

	// Use transaction for atomicity
	tx, err := Mdb.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("DeleteComment: failed to begin transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Delete comment (CASCADE will handle replies)
	_, err = tx.ExecContext(ctx, "DELETE FROM comments WHERE comment_id = $1", commentID)
	if err != nil {
		log.Printf("DeleteComment: failed to delete comment: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete comment")
		return
	}

	// Update video comment count
	_, err = tx.ExecContext(ctx,
		"UPDATE videos SET video_comments = video_comments - 1 WHERE video_id = $1",
		commentedTo,
	)
	if err != nil {
		log.Printf("DeleteComment: failed to update video comment count: %v", err)
		// Don't fail the request, just log the error
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("DeleteComment: failed to commit transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to complete deletion")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Comment deleted successfully"})
}

// DeleteReply deletes a reply by replyID (admin only)
func DeleteReply(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	replyID := chi.URLParam(r, "replyID")
	if replyID == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Reply ID is required")
		return
	}

	// Check if reply exists and get comment_id
	var repliedTo string
	err := Mdb.DB.QueryRowContext(ctx,
		"SELECT replied_to FROM replies WHERE reply_id = $1",
		replyID,
	).Scan(&repliedTo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "Reply not found")
		} else {
			log.Printf("DeleteReply: failed to fetch reply: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load reply")
		}
		return
	}

	// Use transaction for atomicity
	tx, err := Mdb.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("DeleteReply: failed to begin transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Delete reply
	_, err = tx.ExecContext(ctx, "DELETE FROM replies WHERE reply_id = $1", replyID)
	if err != nil {
		log.Printf("DeleteReply: failed to delete reply: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete reply")
		return
	}

	// Update comment reply count
	_, err = tx.ExecContext(ctx,
		"UPDATE comments SET total_replies = total_replies - 1 WHERE comment_id = $1",
		repliedTo,
	)
	if err != nil {
		log.Printf("DeleteReply: failed to update comment reply count: %v", err)
		// Don't fail the request, just log the error
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("DeleteReply: failed to commit transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to complete deletion")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Reply deleted successfully"})
}

// GetCounters returns aggregated counters for all entities (admin only)
// Uses dedicated system_counters table maintained by database triggers
// Provides instant, 100% accurate counts without scanning large tables
func GetCounters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	var counters struct {
		Users    int `json:"users"`
		Videos   int `json:"videos"`
		Comments int `json:"comments"`
		Replies  int `json:"replies"`
		Upvotes  int `json:"upvotes"`
		Downvotes int `json:"downvotes"`
		Views    int `json:"views"`
		UpdatedAt time.Time `json:"updated_at,omitempty"`
	}

	// Query the dedicated counter table - instant read from a single row
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT users_count, videos_count, comments_count, replies_count, 
			upvotes_count, downvotes_count, views_count, updated_at
		FROM system_counters WHERE id = 1`,
	).Scan(
		&counters.Users,
		&counters.Videos,
		&counters.Comments,
		&counters.Replies,
		&counters.Upvotes,
		&counters.Downvotes,
		&counters.Views,
		&counters.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("GetCounters: system_counters table not initialized, run migration 008")
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Counters not initialized. Please run database migrations.")
			return
		}
		log.Printf("GetCounters: failed to query counters: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch counters")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"counters": counters,
	})
}

// ResyncCounters manually resyncs counters with actual table counts (admin only)
// Useful if counters get out of sync due to direct database operations or trigger failures
func ResyncCounters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := requireAdmin(w, r)
	if !ok {
		return
	}

	// Use transaction to ensure atomicity
	tx, err := Mdb.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("ResyncCounters: failed to begin transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Update counters with actual counts from tables
	// Use INSERT ... ON CONFLICT to ensure row exists, then UPDATE
	_, err = tx.ExecContext(ctx,
		`INSERT INTO system_counters (id, users_count, videos_count, comments_count, replies_count, upvotes_count, downvotes_count, views_count)
		VALUES (1, 0, 0, 0, 0, 0, 0, 0)
		ON CONFLICT (id) DO NOTHING`,
	)
	if err != nil {
		log.Printf("ResyncCounters: failed to ensure counter row exists: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to initialize counters")
		return
	}

	// Now update with actual counts
	_, err = tx.ExecContext(ctx,
		`UPDATE system_counters SET
			users_count = (SELECT COUNT(*) FROM users),
			videos_count = (SELECT COUNT(*) FROM videos),
			comments_count = (SELECT COUNT(*) FROM comments),
			replies_count = (SELECT COUNT(*) FROM replies),
			upvotes_count = (SELECT COUNT(*) FROM upvotes),
			downvotes_count = (SELECT COUNT(*) FROM downvotes),
			views_count = (SELECT COUNT(*) FROM views),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`,
	)
	if err != nil {
		log.Printf("ResyncCounters: failed to resync counters: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to resync counters")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ResyncCounters: failed to commit transaction: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to complete resync")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Counters resynced successfully"})
}
