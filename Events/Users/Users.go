package users

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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"

	Search "hifi/Events/Search"
	Auth "hifi/Services/Auth"
	Mdb "hifi/Services/Mdb"
	storage "hifi/Services/Storage"
	Utils "hifi/Utils"
)

var GetClaims func(r *http.Request) (*Auth.Token, bool) = Auth.GetClaims

// Handle sets up the routes for user endpoints
func Handle(r chi.Router) {
	r.Get("/{username}", GetUser)
	r.Get("/self", GetSelf)
	r.Delete("/{username}", DeleteUser)
	r.Put("/self", UpdateUser)
	r.Get("/availability/{username}", UsernameAvailability)
	r.Get("/list", ListUser) // Added route for ListUser
	r.Post("/profile-photo/upload", UploadProfilePhoto)
}

// GetUser retrieves a user by username
func GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := GetClaims(r)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	username := chi.URLParam(r, "username")
	if username == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Normalize username
	username = strings.ToLower(strings.TrimSpace(username))

	// Fetch user (excluding admin users at database level for efficiency)
	var user User
	var bioNull, emailNull sql.NullString
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT id, uid, username, name, role, profile_picture, bio, email, 
			followers, following, total_streams, total_videos, created_at, updated_at
		FROM users WHERE username = $1 AND role != 'admin'`,
		username,
	).Scan(
		&user.ID, &user.UID, &user.Username, &user.Name, &user.Role,
		&user.ProfilePicture, &bioNull, &emailNull, &user.Followers, &user.Following,
		&user.TotalStreams, &user.TotalVideos, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("GetUser: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return
	}
	user.Bio = nullStringToPtr(bioNull)
	user.Email = nullStringToPtr(emailNull)

	// Check if authenticated user follows this user
	following := false
	var hasFollow bool
	err = Mdb.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM followers WHERE followed_by = $1 AND followed_to = $2)`,
		claims.UID, user.UID,
	).Scan(&hasFollow)
	if err == nil {
		following = hasFollow
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"user":      user,
		"following": following,
	})
}

// GetSelf retrieves the authenticated user's own profile
// DEPRECATED: This endpoint is deprecated. Use GET /users/{username} with your own username instead.
func GetSelf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := GetClaims(r)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := fetchUserByUID(ctx, claims.UID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("GetSelf: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user")
		}
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{"user": user})
}

// DeleteUser deletes a user account (soft delete to deleted_users table)
// Uses transaction to ensure atomicity
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := GetClaims(r)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	username := chi.URLParam(r, "username")
	if username == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Normalize username
	username = strings.ToLower(strings.TrimSpace(username))

	// Fetch user
	existing, err := fetchUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("DeleteUser: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load user")
		}
		return
	}

	// Only allow users to delete themselves
	if claims.UID != existing.UID {
		Utils.SendErrorResponse(w, http.StatusForbidden, "Forbidden: you can only delete your own account")
		return
	}

	// Note: With custom JWT auth, user deletion is handled by removing from database only
	// No separate auth service deletion needed

	// Use transaction for atomicity
	// Foreign key CASCADE will handle database cleanup automatically:
	// - All videos will be deleted
	// - All video_on_upload records will be deleted
	// - All followers relationships will be deleted
	// - All blocklists relationships will be deleted
	// - All upvotes, downvotes, comments, replies, views will be deleted
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

	// Delete from users
	// Foreign key CASCADE will automatically delete:
	// - All videos (videos table)
	// - All video_on_upload records
	// - All followers relationships (both followed_by and followed_to)
	// - All blocklists relationships (both blocked_by and blocked_to)
	// - All upvotes, downvotes, comments, replies, views
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

// UpdateUser updates user information
// Users can only update their own account
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := GetClaims(r)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Fetch existing user from token (like GetSelf)
	existing, err := fetchUserByUID(ctx, claims.UID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Utils.SendErrorResponse(w, http.StatusNotFound, "User not found")
		} else {
			log.Printf("UpdateUser: failed to fetch user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load user")
		}
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("UpdateUser: failed to read body: %v", err)
		Utils.SendErrorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var payload struct {
		Name           *string `json:"name"`
		ProfilePicture *string `json:"profile_picture"`
		Role           *string `json:"role"`
		Bio            *string `json:"bio"`
		Email          *string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	// Validate and process name update
	if payload.Name != nil {
		name := strings.TrimSpace(*payload.Name)
		if err := ValidateName(name); err != nil {
			Utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, name)
		argPos++
	}

	// Process profile picture update (no validation needed, just sanitize)
	if payload.ProfilePicture != nil {
		pic := strings.TrimSpace(*payload.ProfilePicture)
		updates = append(updates, fmt.Sprintf("profile_picture = $%d", argPos))
		args = append(args, pic)
		argPos++
	}

	// Validate and process role update (only allows "user" or "creator", not "admin")
	if payload.Role != nil {
		role := strings.ToLower(strings.TrimSpace(*payload.Role))
		if err := ValidateRole(role); err != nil {
			Utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		updates = append(updates, fmt.Sprintf("role = $%d", argPos))
		args = append(args, role)
		argPos++
	}

	// Validate and process bio update
	if payload.Bio != nil {
		bio := strings.TrimSpace(*payload.Bio)
		if err := ValidateBio(bio); err != nil {
			Utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		// Store as NULL if empty, otherwise store the value
		if bio == "" {
			updates = append(updates, "bio = NULL")
		} else {
			updates = append(updates, fmt.Sprintf("bio = $%d", argPos))
			args = append(args, bio)
			argPos++
		}
	}

	// Validate and process email update
	if payload.Email != nil {
		email := strings.TrimSpace(*payload.Email)
		if email == "" {
			// Set to NULL if empty string
			updates = append(updates, "email = NULL")
		} else {
			if err := ValidateEmail(email); err != nil {
				Utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
			// Check if email is already taken by another user
			var existingUID string
			err := Mdb.DB.QueryRowContext(ctx,
				"SELECT uid FROM users WHERE email = $1 AND uid != $2",
				email, existing.UID,
			).Scan(&existingUID)
			if err == nil {
				Utils.SendErrorResponse(w, http.StatusConflict, "Email already in use")
				return
			} else if !errors.Is(err, sql.ErrNoRows) {
				log.Printf("UpdateUser: failed to check email: %v", err)
				Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to check email availability")
				return
			}
			updates = append(updates, fmt.Sprintf("email = $%d", argPos))
			args = append(args, email)
			argPos++
		}
	}

	// If no updates, return current user
	if len(updates) == 0 {
		Utils.SendSuccessResponse(w, map[string]interface{}{"user": existing})
		return
	}

	// Add updated_at and uid for WHERE clause
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++
	args = append(args, existing.UID)

	// Execute update
	query := "UPDATE users SET " + strings.Join(updates, ", ") + fmt.Sprintf(" WHERE uid = $%d", argPos)
	_, err = Mdb.DB.ExecContext(ctx, query, args...)
	if err != nil {
		log.Printf("UpdateUser: failed to update user: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Fetch updated user
	updatedUser, err := fetchUserByUID(ctx, existing.UID)
	if err != nil {
		log.Printf("UpdateUser: failed to fetch updated user: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load updated user")
		return
	}

	// Update user in Elasticsearch if profile_picture was changed (non-blocking)
	if payload.ProfilePicture != nil {
		go func() {
			esCtx := context.Background()
			if err := Search.IndexUser(esCtx, updatedUser.UID, updatedUser.Username, updatedUser.ProfilePicture); err != nil {
				log.Printf("UpdateUser: failed to update user in Elasticsearch: %v", err)
			}
		}()
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{"user": updatedUser})
}

// UsernameAvailability checks if a username is available
// TODO: Add rate limiting middleware to prevent abuse
func UsernameAvailability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "username")))

	if username == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Username is required")
		return
	}

	// Validate username format
	if err := ValidateUsername(username); err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	exists, err := CheckUsernameExists(ctx, username)
	if err != nil {
		log.Printf("UsernameAvailability: failed to check username: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Error checking username availability")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"available": !exists,
		"username":  username,
	})
}

// ListUser lists all users with pagination
// Query params: ?limit=20&offset=0
func ListUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := GetClaims(r)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	seedStr := r.URL.Query().Get("seed")

	limit := DefaultPageLimit
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= MaxPageLimit {
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
		seed = "hifi_users_shuffle_2024" // Default seed for stable shuffle
	}
	rows, err := Mdb.DB.QueryContext(ctx,
		`SELECT id, uid, username, name, role, profile_picture, bio, email, 
			followers, following, total_streams, total_videos, created_at, updated_at
		FROM users 
		WHERE role != 'admin'
		ORDER BY hashtext(id::text || $1)
		LIMIT $2 OFFSET $3`,
		seed, limit, offset,
	)
	if err != nil {
		log.Printf("ListUser: failed to query users: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer rows.Close()

	var users []User
	var userUIDs []string
	for rows.Next() {
		var user User
		var bioNull, emailNull sql.NullString
		if err := rows.Scan(
			&user.ID, &user.UID, &user.Username, &user.Name, &user.Role,
			&user.ProfilePicture, &bioNull, &emailNull, &user.Followers, &user.Following,
			&user.TotalStreams, &user.TotalVideos, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			log.Printf("ListUser: failed to scan user: %v", err)
			Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch users")
			return
		}
		user.Bio = nullStringToPtr(bioNull)
		user.Email = nullStringToPtr(emailNull)
		users = append(users, user)
		userUIDs = append(userUIDs, user.UID)
	}

	if err := rows.Err(); err != nil {
		log.Printf("ListUser: row iteration error: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to iterate users")
		return
	}

	// Check which users the authenticated user follows
	followingMap := make(map[string]bool)
	if len(userUIDs) > 0 {
		// Build query to check all follows at once using ANY
		followRows, err := Mdb.DB.QueryContext(ctx,
			`SELECT followed_to FROM followers 
			WHERE followed_by = $1 AND followed_to = ANY($2)`,
			claims.UID, pq.Array(userUIDs),
		)
		if err != nil {
			log.Printf("ListUser: failed to query following status: %v", err)
			// Continue without following data rather than failing
		} else {
			defer followRows.Close()
			for followRows.Next() {
				var followedUID string
				if err := followRows.Scan(&followedUID); err == nil {
					followingMap[followedUID] = true
				}
			}
		}
	}

	// Build response with following status for each user
	type UserWithFollowing struct {
		User      User `json:"user"`
		Following bool `json:"following"`
	}

	usersWithFollowing := make([]UserWithFollowing, len(users))
	for i, user := range users {
		usersWithFollowing[i] = UserWithFollowing{
			User:      user,
			Following: followingMap[user.UID],
		}
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"users":  usersWithFollowing,
		"limit":  limit,
		"offset": offset,
		"count":  len(users),
		"seed":   seed,
	})
}

// UploadProfilePhoto generates a presigned URL for uploading a profile photo
func UploadProfilePhoto(w http.ResponseWriter, r *http.Request) {
	claims, ok := GetClaims(r)
	if !ok {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Generate the profile photo path
	profilePhotoPath := fmt.Sprintf("ProfileProto/users/%s.jpg", claims.UID)

	// Generate presigned upload URL (20 minute expiry)
	gatewayURL, err := storage.GeneratePresignedUploadURL(profilePhotoPath, 20*time.Minute)
	if err != nil {
		log.Printf("UploadProfilePhoto: failed to generate presigned upload URL: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to generate presigned upload URL")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"message":     "Profile photo upload URL generated",
		"gateway_url": gatewayURL,
		"path":        profilePhotoPath,
	})
}
