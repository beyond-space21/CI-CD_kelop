package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	Search "hifi/Events/Search"
	Users "hifi/Events/Users"
	AuthService "hifi/Services/Auth"
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

// Handle sets up the routes for authentication endpoints
func Handle(r chi.Router) {
	r.Post("/register", Register)
	r.Post("/login", Login)
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Register creates a new user account and returns a JWT token
func Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Register: failed to read body: %v", err)
		Utils.SendErrorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var input RegisterRequest
	if err := json.Unmarshal(body, &input); err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate inputs
	if err := Users.ValidateUsername(input.Username); err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := Users.ValidateName(input.Name); err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.Password == "" || len(input.Password) < 6 {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	// Normalize username
	username := strings.ToLower(strings.TrimSpace(input.Username))
	name := strings.TrimSpace(input.Name)

	// Check if username is already taken
	exists, err := Users.CheckUsernameExists(ctx, username)
	if err != nil {
		log.Printf("Register: failed to check username: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to check username availability")
		return
	}
	if exists {
		Utils.SendErrorResponse(w, http.StatusConflict, "username already in use")
		return
	}

	// Hash password
	passwordHash, err := AuthService.HashPassword(input.Password)
	if err != nil {
		log.Printf("Register: failed to hash password: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	// Generate UID (using username + timestamp for uniqueness)
	uid := Users.GenerateUID(username)

	// Insert new user
	now := time.Now()
	var userID int
	err = Mdb.DB.QueryRowContext(ctx,
		`INSERT INTO users (uid, username, name, role, password_hash, profile_picture, followers, following, 
			total_streams, total_videos, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		uid, username, name, Users.DefaultRole, passwordHash, "", 0, 0, 0, 0, now, now,
	).Scan(&userID)
	if err != nil {
		log.Printf("Register: failed to insert user: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Generate JWT token
	token, err := AuthService.GenerateToken(uid)
	if err != nil {
		log.Printf("Register: failed to generate token: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to generate authentication token")
		return
	}

	// Index user in Elasticsearch (non-blocking, log errors but don't fail registration)
	go func() {
		esCtx := context.Background()
		if err := Search.IndexUser(esCtx, uid, username, ""); err != nil {
			log.Printf("Register: failed to index user in Elasticsearch: %v", err)
		}
	}()

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"id":         userID,
		"uid":        uid,
		"token":      token,
		"expires_in": int(AuthService.TokenValidity.Seconds()),
		"user": map[string]interface{}{
			"uid":      uid,
			"username": username,
			"name":     name,
		},
	})
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login authenticates a user and returns a JWT token
func Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Login: failed to read body: %v", err)
		Utils.SendErrorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var input LoginRequest
	if err := json.Unmarshal(body, &input); err != nil {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Username == "" || input.Password == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "username and password are required")
		return
	}

	// Normalize username
	username := strings.ToLower(strings.TrimSpace(input.Username))

	// Fetch user by username
	var user Users.User
	var passwordHash string
	var bioNull, emailNull sql.NullString
	err = Mdb.DB.QueryRowContext(ctx,
		`SELECT id, uid, username, name, role, password_hash, profile_picture, bio, email, 
			followers, following, total_streams, total_videos, created_at, updated_at
		FROM users WHERE username = $1`,
		username,
	).Scan(
		&user.ID, &user.UID, &user.Username, &user.Name, &user.Role,
		&passwordHash, &user.ProfilePicture, &bioNull, &emailNull, &user.Followers, &user.Following,
		&user.TotalStreams, &user.TotalVideos, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			Utils.SendErrorResponse(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		log.Printf("Login: failed to fetch user: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}
	user.Bio = nullStringToPtr(bioNull)
	user.Email = nullStringToPtr(emailNull)

	// Verify password
	if !AuthService.CheckPasswordHash(input.Password, passwordHash) {
		Utils.SendErrorResponse(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	// Generate JWT token
	token, err := AuthService.GenerateToken(user.UID)
	if err != nil {
		log.Printf("Login: failed to generate token: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to generate authentication token")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"token":      token,
		"expires_in": int(AuthService.TokenValidity.Seconds()),
		"user": map[string]interface{}{
			"uid":      user.UID,
			"username": user.Username,
			"name":     user.Name,
		},
	})
}
