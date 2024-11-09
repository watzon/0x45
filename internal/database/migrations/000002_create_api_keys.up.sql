CREATE TABLE api_keys (
    key VARCHAR(64) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    
    max_file_size BIGINT DEFAULT 10485760,
    max_expiration VARCHAR(32) DEFAULT '24h',
    rate_limit INTEGER DEFAULT 100,
    allow_private BOOLEAN DEFAULT TRUE,
    allow_updates BOOLEAN DEFAULT FALSE,
    
    email VARCHAR(255),
    name VARCHAR(255),
    
    last_used_at TIMESTAMP,
    usage_count BIGINT DEFAULT 0,
    
    allow_shortlinks BOOLEAN DEFAULT FALSE,
    shortlink_quota INTEGER DEFAULT 0,
    shortlink_prefix VARCHAR(16)
);

CREATE INDEX idx_api_keys_deleted_at ON api_keys(deleted_at); 