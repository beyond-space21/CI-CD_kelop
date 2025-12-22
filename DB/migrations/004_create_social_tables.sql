-- Create followers table
CREATE TABLE IF NOT EXISTS followers (
    id SERIAL PRIMARY KEY,
    followed_by VARCHAR(255) NOT NULL, -- User who is following
    followed_to VARCHAR(255) NOT NULL, -- User who is followed
    followed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(followed_by, followed_to)
);

CREATE INDEX IF NOT EXISTS idx_followers_followed_by ON followers(followed_by);
CREATE INDEX IF NOT EXISTS idx_followers_followed_to ON followers(followed_to);

-- Create blocklists table
CREATE TABLE IF NOT EXISTS blocklists (
    id SERIAL PRIMARY KEY,
    blocked_by VARCHAR(255) NOT NULL, -- User who blocked
    blocked_to VARCHAR(255) NOT NULL, -- User who is blocked
    blocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(blocked_by, blocked_to)
);

CREATE INDEX IF NOT EXISTS idx_blocklists_blocked_by ON blocklists(blocked_by);
CREATE INDEX IF NOT EXISTS idx_blocklists_blocked_to ON blocklists(blocked_to);

-- Create upvotes table
CREATE TABLE IF NOT EXISTS upvotes (
    id SERIAL PRIMARY KEY,
    upvoted_by VARCHAR(255) NOT NULL, -- User who upvoted
    upvoted_to VARCHAR(255) NOT NULL, -- Content which is upvoted
    upvoted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(upvoted_by, upvoted_to)
);

CREATE INDEX IF NOT EXISTS idx_upvotes_upvoted_by ON upvotes(upvoted_by);
CREATE INDEX IF NOT EXISTS idx_upvotes_upvoted_to ON upvotes(upvoted_to);

-- Create downvotes table
CREATE TABLE IF NOT EXISTS downvotes (
    id SERIAL PRIMARY KEY,
    downvoted_by VARCHAR(255) NOT NULL, -- User who downvoted
    downvoted_to VARCHAR(255) NOT NULL, -- Video which is downvoted
    downvoted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(downvoted_by, downvoted_to)
);

CREATE INDEX IF NOT EXISTS idx_downvotes_downvoted_by ON downvotes(downvoted_by);
CREATE INDEX IF NOT EXISTS idx_downvotes_downvoted_to ON downvotes(downvoted_to);

-- Create comments table
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    comment_id VARCHAR(255) UNIQUE NOT NULL,
    commented_by VARCHAR(255) NOT NULL, -- User who commented
    commented_to VARCHAR(255) NOT NULL, -- Video which is commented
    commented_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    comment TEXT NOT NULL,
    comment_by_username VARCHAR(255) NOT NULL,
    total_replies INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_comments_comment_id ON comments(comment_id);
CREATE INDEX IF NOT EXISTS idx_comments_commented_by ON comments(commented_by);
CREATE INDEX IF NOT EXISTS idx_comments_commented_to ON comments(commented_to);

-- Create replies table
CREATE TABLE IF NOT EXISTS replies (
    id SERIAL PRIMARY KEY,
    reply_id VARCHAR(255) UNIQUE NOT NULL,
    replied_by VARCHAR(255) NOT NULL, -- User who replied
    replied_to VARCHAR(255) NOT NULL, -- Comment which is replied
    replied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reply TEXT NOT NULL,
    reply_by_username VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_replies_reply_id ON replies(reply_id);
CREATE INDEX IF NOT EXISTS idx_replies_replied_by ON replies(replied_by);
CREATE INDEX IF NOT EXISTS idx_replies_replied_to ON replies(replied_to);

-- Create views table
CREATE TABLE IF NOT EXISTS views (
    id SERIAL PRIMARY KEY,
    viewed_by VARCHAR(255) NOT NULL, -- User who viewed
    viewed_to VARCHAR(255) NOT NULL, -- Content which is viewed
    viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(viewed_by, viewed_to)
);

CREATE INDEX IF NOT EXISTS idx_views_viewed_by ON views(viewed_by);
CREATE INDEX IF NOT EXISTS idx_views_viewed_to ON views(viewed_to);

