package users

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	Mdb "hifi/Services/Mdb"
)

// nullStringToPtr converts sql.NullString to *string (nil if NULL, pointer to value if not)
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Constants for validation
const (
	MinUsernameLength = 3
	MaxUsernameLength = 30
	DefaultPageLimit  = 20
	MaxPageLimit      = 100
	DefaultRole       = "user" // Default role for new users
	UserRole          = "user"
	CreatorRole       = "creator"
)

var (
	// usernameRegex validates username: 3-30 chars, alphanumeric + underscores, lowercase
	usernameRegex = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)
)

// fetchUserByUID retrieves a user by their UID
// Ensure INDEX on users(uid) for optimal performance
func fetchUserByUID(ctx context.Context, uid string) (*User, error) {
	var user User
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

// fetchUserByUsername retrieves a user by their username
// Ensure INDEX on users(username) for optimal performance
func fetchUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	var bioNull, emailNull sql.NullString
	err := Mdb.DB.QueryRowContext(ctx,
		`SELECT id, uid, username, name, role, profile_picture, bio, email, 
			followers, following, total_streams, total_videos, created_at, updated_at
		FROM users WHERE username = $1`,
		username,
	).Scan(
		&user.ID, &user.UID, &user.Username, &user.Name, &user.Role,
		&user.ProfilePicture, &bioNull, &emailNull, &user.Followers, &user.Following,
		&user.TotalStreams, &user.TotalVideos, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("fetchUserByUsername: %w", err)
	}
	user.Bio = nullStringToPtr(bioNull)
	user.Email = nullStringToPtr(emailNull)
	return &user, nil
}

// ValidateUsername checks if username meets requirements (exported for use in Auth package)
func ValidateUsername(username string) error {
	username = strings.TrimSpace(strings.ToLower(username))
	if username == "" {
		return errors.New("username is required")
	}
	if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
		return fmt.Errorf("username must be between %d and %d characters", MinUsernameLength, MaxUsernameLength)
	}
	if !usernameRegex.MatchString(username) {
		return errors.New("username must contain only lowercase letters, numbers, and underscores")
	}
	return nil
}

// ValidateName checks if name is valid (exported for use in Auth package)
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if len(name) > MaxUsernameLength {
		return errors.New("name must be less than " + strconv.Itoa(MaxUsernameLength) + " characters")
	}
	return nil
}

// ValidateRole checks if role is valid (exported for use in Users package)
// Only allows "user" or "creator", not "admin"
func ValidateRole(role string) error {
	role = strings.TrimSpace(strings.ToLower(role))
	if role != UserRole && role != CreatorRole {
		return errors.New("role must be either 'user' or 'creator'")
	}
	return nil
}

// ValidateEmail checks if email is valid (exported for use in Users package)
func ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil // Email is optional, empty is valid
	}
	// Basic email validation regex
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	if len(email) > 255 {
		return errors.New("email must be less than 255 characters")
	}
	return nil
}

// ValidateBio checks if bio is valid (exported for use in Users package)
func ValidateBio(bio string) error {
	// Bio is optional, but if provided, check length
	if len(bio) > 1000 {
		return errors.New("bio must be less than 1000 characters")
	}
	return nil
}


// CheckUsernameExists checks if a username is already taken (exported for use in Auth package)
func CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	var id int
	err := Mdb.DB.QueryRowContext(ctx,
		"SELECT id FROM users WHERE username = $1",
		username,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checkUsernameExists: %w", err)
	}
	return true, nil
}

// GenerateUID generates a unique user ID based on username and timestamp
func GenerateUID(username string) string {
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("%s-%d", username, timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:32] // Use first 32 chars of hash
}

