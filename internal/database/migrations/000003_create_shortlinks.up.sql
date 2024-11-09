CREATE TABLE shortlinks (
    id VARCHAR(8) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    
    target_url TEXT NOT NULL,
    title VARCHAR(255),
    
    api_key VARCHAR(64) NOT NULL,
    delete_key VARCHAR(32) NOT NULL,
    expires_at TIMESTAMP,
    
    clicks BIGINT DEFAULT 0,
    last_click TIMESTAMP,
    
    metadata JSON
);

CREATE INDEX idx_shortlinks_deleted_at ON shortlinks(deleted_at);
CREATE INDEX idx_shortlinks_expires_at ON shortlinks(expires_at);
CREATE INDEX idx_shortlinks_api_key ON shortlinks(api_key); 