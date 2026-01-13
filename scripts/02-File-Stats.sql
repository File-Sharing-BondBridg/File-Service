CREATE TABLE IF NOT EXISTS user_file_stats (
    user_id UUID PRIMARY KEY,
    file_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
