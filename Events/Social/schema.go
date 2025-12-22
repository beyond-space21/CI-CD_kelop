package social

import "time"

type Followers struct {
	ID               int       `db:"id" json:"-"`
	FollowedBy       string    `db:"followed_by" json:"followed_by"`             // UID of user who is following
	FollowedByUsername string  `db:"followed_by_username" json:"followed_by_username,omitempty"` // Username of user who is following (for responses)
	FollowedTo       string    `db:"followed_to" json:"followed_to"`             // UID of user who is followed
	FollowedToUsername string  `db:"followed_to_username" json:"followed_to_username,omitempty"` // Username of user who is followed (for responses)
	FollowedAt       time.Time `db:"followed_at" json:"followed_at"`
}

type Blocklists struct {
	ID        int       `db:"id" json:"-"`
	BlockedBy string    `db:"blocked_by" json:"blocked_by"` // User who blocked
	BlockedTo string    `db:"blocked_to" json:"blocked_to"` // User who is blocked
	BlockedAt time.Time `db:"blocked_at" json:"blocked_at"`
}

type Upvotes struct {
	ID        int       `db:"id" json:"-"`
	UpvotedBy string    `db:"upvoted_by" json:"upvoted_by"` // User who upvoted
	UpvotedTo string    `db:"upvoted_to" json:"upvoted_to"` // Content which is upvoted
	UpvotedAt time.Time `db:"upvoted_at" json:"upvoted_at"`
}

type Downvotes struct {
	ID          int       `db:"id" json:"-"`
	DownvotedBy string    `db:"downvoted_by" json:"downvoted_by"` // User who downvoted
	DownvotedTo string    `db:"downvoted_to" json:"downvoted_to"` // video which is downvoted
	DownvotedAt time.Time `db:"downvoted_at" json:"downvoted_at"`
}

type Comments struct {
	ID               int       `db:"id" json:"-"`
	CommentID        string    `db:"comment_id" json:"comment_id"`     // Comment ID
	CommentedBy      string    `db:"commented_by" json:"commented_by"` // User who commented
	CommentedTo      string    `db:"commented_to" json:"commented_to"` // video which is commented
	CommentedAt      time.Time `db:"commented_at" json:"commented_at"`
	Comment          string    `db:"comment" json:"comment"`
	CommentByUsername string   `db:"comment_by_username" json:"comment_by_username"`
	TotalReplies     int       `db:"total_replies" json:"total_replies"`
}

type Replies struct {
	ID             int       `db:"id" json:"-"`
	ReplyID        string    `db:"reply_id" json:"reply_id"`     // Reply ID
	RepliedBy      string    `db:"replied_by" json:"replied_by"` // User who replied
	RepliedTo      string    `db:"replied_to" json:"replied_to"` // Comment which is replied
	RepliedAt      time.Time `db:"replied_at" json:"replied_at"`
	Reply          string    `db:"reply" json:"reply"`
	ReplyByUsername string   `db:"reply_by_username" json:"reply_by_username"`
}

type Views struct {
	ID       int       `db:"id" json:"-"`
	ViewedBy string    `db:"viewed_by" json:"viewed_by"` // User who viewed
	ViewedTo string    `db:"viewed_to" json:"viewed_to"` // Content which is viewed
	ViewedAt time.Time `db:"viewed_at" json:"viewed_at"`
}
