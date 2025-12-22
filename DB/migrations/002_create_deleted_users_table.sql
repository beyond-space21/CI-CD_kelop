-- Create deleted_users table (same structure as users)
CREATE TABLE IF NOT EXISTS deleted_users (
    id SERIAL PRIMARY KEY,
    uid VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    profile_picture TEXT,
    followers INTEGER DEFAULT 0,
    following INTEGER DEFAULT 0,
    total_streams INTEGER DEFAULT 0,
    total_videos INTEGER DEFAULT 0,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

