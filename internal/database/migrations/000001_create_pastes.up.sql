CREATE TABLE pastes (
    id VARCHAR(16) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    
    filename VARCHAR(255),
    mime_type VARCHAR(255),
    size BIGINT,
    extension VARCHAR(32),
    
    storage_path VARCHAR(512),
    storage_type VARCHAR(32),
    storage_name VARCHAR(64),
    
    private BOOLEAN DEFAULT FALSE,
    delete_key VARCHAR(32) NOT NULL,
    api_key VARCHAR(64),
    
    expires_at TIMESTAMP,
    
    metadata JSON
);

CREATE INDEX idx_pastes_deleted_at ON pastes(deleted_at);
CREATE INDEX idx_pastes_expires_at ON pastes(expires_at);
CREATE INDEX idx_pastes_api_key ON pastes(api_key); 