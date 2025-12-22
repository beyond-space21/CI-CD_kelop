package users

import "time"

// User represents a user in the system
// Ensure INDEXES on: users(uid), users(username), users(email) for optimal query performance
type User struct {
	ID             int        `db:"id" json:"id"`
	UID            string     `db:"uid" json:"uid"`
	Username       string     `db:"username" json:"username"`
	Name           string     `db:"name" json:"name"`
	Role           string     `db:"role" json:"role"` // Default value: "user"
	ProfilePicture string     `db:"profile_picture" json:"profile_picture"`
	Bio            *string    `db:"bio" json:"bio,omitempty"`           // Optional biography (nullable)
	Email          *string    `db:"email" json:"email,omitempty"`       // Optional email address (nullable)
	Followers      int        `db:"followers" json:"followers"`
	Following      int        `db:"following" json:"following"`
	TotalStreams   int        `db:"total_streams" json:"total_streams"`
	TotalVideos    int        `db:"total_videos" json:"total_videos"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}
