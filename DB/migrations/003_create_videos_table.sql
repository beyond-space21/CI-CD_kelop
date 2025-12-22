-- Create videos table
CREATE TABLE IF NOT EXISTS videos (
    id SERIAL PRIMARY KEY,
    video_id VARCHAR(255) UNIQUE NOT NULL,
    video_url TEXT NOT NULL,
    video_thumbnail TEXT,
    video_title VARCHAR(500),
    video_description TEXT,
    video_tags TEXT[], -- Array of strings
    video_views INTEGER DEFAULT 0,
    video_upvotes INTEGER DEFAULT 0,
    video_downvotes INTEGER DEFAULT 0,
    video_comments INTEGER DEFAULT 0,
    user_uid VARCHAR(255) NOT NULL,
    user_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_videos_video_id ON videos(video_id);
CREATE INDEX IF NOT EXISTS idx_videos_user_uid ON videos(user_uid);
CREATE INDEX IF NOT EXISTS idx_videos_user_username ON videos(user_username);

-- Create video_on_upload table (same structure as videos)
CREATE TABLE IF NOT EXISTS video_on_upload (
    id SERIAL PRIMARY KEY,
    video_id VARCHAR(255) UNIQUE NOT NULL,
    video_url TEXT NOT NULL,
    video_thumbnail TEXT,
    video_title VARCHAR(500),
    video_description TEXT,
    video_tags TEXT[],
    video_views INTEGER DEFAULT 0,
    video_upvotes INTEGER DEFAULT 0,
    video_downvotes INTEGER DEFAULT 0,
    video_comments INTEGER DEFAULT 0,
    user_uid VARCHAR(255) NOT NULL,
    user_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_video_on_upload_video_id ON video_on_upload(video_id);

