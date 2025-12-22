package social

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	Auth "hifi/Services/Auth"
	Mdb "hifi/Services/Mdb"
	Utils "hifi/Utils"
)

func HandleUsers(req chi.Router) {
	req.Post("/follow/{username}", Follow)
	req.Post("/unfollow/{username}", Unfollow)
	req.Get("/followers", ListFollowers)
	req.Get("/following", ListFollowing)
}

func Follow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	username := chi.URLParam(r, "username")
	if username == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Get the UID of the user to follow
	var userUID string
	err := Mdb.DB.QueryRowContext(ctx,
		"SELECT uid FROM users WHERE username = $1",
		username,
	).Scan(&userUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("Follow: failed to fetch user to follow: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return
	}

	// Prevent self-follow
	if claims.UID == userUID {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "You cannot follow yourself")
		return
	}

	// Check if already following
	var existingID int
	err = Mdb.DB.QueryRowContext(ctx,
		"SELECT id FROM followers WHERE followed_by = $1 AND followed_to = $2",
		claims.UID, userUID,
	).Scan(&existingID)
	if err == nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "You are already following this user")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("Follow: failed to check existing follow: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to check existing follow")
		return
	}

	// Insert follower relationship (using UIDs)
	_, err = Mdb.DB.ExecContext(ctx,
		"INSERT INTO followers (followed_by, followed_to, followed_at) VALUES ($1, $2, $3)",
		claims.UID, userUID, time.Now(),
	)
	if err != nil {
		log.Printf("Follow: failed to insert follower: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to insert follower")
		return
	}

	// Update follower counts
	_, err = Mdb.DB.ExecContext(ctx, "UPDATE users SET followers = followers + 1 WHERE uid = $1", userUID)
	if err != nil {
		log.Printf("Follow: failed to update followers: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update followers")
		return
	}

	_, err = Mdb.DB.ExecContext(ctx, "UPDATE users SET following = following + 1 WHERE uid = $1", claims.UID)
	if err != nil {
		log.Printf("Follow: failed to update following: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update following")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Followed successfully"})
}

func Unfollow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	username := chi.URLParam(r, "username")
	if username == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Get the UID of the user to unfollow
	var userUID string
	err := Mdb.DB.QueryRowContext(ctx,
		"SELECT uid FROM users WHERE username = $1",
		username,
	).Scan(&userUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("Unfollow: failed to fetch user to unfollow: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return
	}

	// Delete follower relationship (using UIDs)
	result, err := Mdb.DB.ExecContext(ctx,
		"DELETE FROM followers WHERE followed_by = $1 AND followed_to = $2",
		claims.UID, userUID,
	)
	if err != nil {
		log.Printf("Unfollow: failed to delete follower: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete follower")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Unfollow: failed to check delete result: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to check delete result")
		return
	}
	if rowsAffected == 0 {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "You are not following this user")
		return
	}

	// Update follower counts
	_, err = Mdb.DB.ExecContext(ctx, "UPDATE users SET followers = followers - 1 WHERE uid = $1", userUID)
	if err != nil {
		log.Printf("Unfollow: failed to update followers: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update followers")
		return
	}

	_, err = Mdb.DB.ExecContext(ctx, "UPDATE users SET following = following - 1 WHERE uid = $1", claims.UID)
	if err != nil {
		log.Printf("Unfollow: failed to update following: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update following")
		return
	}

	Utils.SendSuccessResponse(w, map[string]string{"message": "Unfollowed successfully"})
}

func ListFollowers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
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

	// Get followers ordered by timestamp (newest first) with total count in single query
	// Join with users table to get usernames for response
	// Use window function COUNT(*) OVER() to get total count without separate query
	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT f.id, f.followed_by, u1.username as followed_by_username, 
			f.followed_to, u2.username as followed_to_username, f.followed_at,
			COUNT(*) OVER() as total_count
		FROM followers f
		JOIN users u1 ON f.followed_by = u1.uid
		JOIN users u2 ON f.followed_to = u2.uid
		WHERE f.followed_to = $1 
		ORDER BY f.followed_at DESC
		LIMIT $2 OFFSET $3`,
		claims.UID, limit, offset,
	)
	if err != nil {
		log.Printf("ListFollowers: failed to list followers: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to list followers")
		return
	}
	defer rows.Close()

	var followers []Followers
	var count int
	for rows.Next() {
		var follower Followers
		if err := rows.Scan(
			&follower.ID,
			&follower.FollowedBy,
			&follower.FollowedByUsername,
			&follower.FollowedTo,
			&follower.FollowedToUsername,
			&follower.FollowedAt,
			&count, // total_count from window function (same value for all rows)
		); err != nil {
			log.Printf("ListFollowers: failed to scan follower: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode follower")
			return
		}
		followers = append(followers, follower)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListFollowers: failed to iterate followers: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate followers")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"followers": followers,
		"limit":     limit,
		"offset":    offset,
		"count":     count,
	})
}

func ListFollowing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, auth := Auth.GetClaims(r)
	if !auth {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
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

	// Get following ordered by timestamp (newest first) with total count in single query
	// Join with users table to get usernames for response
	// Use window function COUNT(*) OVER() to get total count without separate query
	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT f.id, f.followed_by, u1.username as followed_by_username, 
			f.followed_to, u2.username as followed_to_username, f.followed_at,
			COUNT(*) OVER() as total_count
		FROM followers f
		JOIN users u1 ON f.followed_by = u1.uid
		JOIN users u2 ON f.followed_to = u2.uid
		WHERE f.followed_by = $1 
		ORDER BY f.followed_at DESC
		LIMIT $2 OFFSET $3`,
		claims.UID, limit, offset,
	)
	if err != nil {
		log.Printf("ListFollowing: failed to list following: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to list following")
		return
	}
	defer rows.Close()

	var following []Followers
	var count int
	for rows.Next() {
		var follower Followers
		if err := rows.Scan(
			&follower.ID,
			&follower.FollowedBy,
			&follower.FollowedByUsername,
			&follower.FollowedTo,
			&follower.FollowedToUsername,
			&follower.FollowedAt,
			&count, // total_count from window function (same value for all rows)
		); err != nil {
			log.Printf("ListFollowing: failed to scan follower: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode following")
			return
		}
		following = append(following, follower)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ListFollowing: failed to iterate following: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate following")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"following": following,
		"limit":     limit,
		"offset":    offset,
		"count":     count,
	})
}
