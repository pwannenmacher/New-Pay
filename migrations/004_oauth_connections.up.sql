-- Create oauth_connections table for multi-provider support
CREATE TABLE IF NOT EXISTS oauth_connections (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(100) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint: one provider connection per user per provider
    CONSTRAINT unique_user_provider UNIQUE (user_id, provider),
    -- Unique constraint: one provider_id per provider (can't link same OAuth account to multiple users)
    CONSTRAINT unique_provider_account UNIQUE (provider, provider_id)
);

-- Create indexes for faster lookups
CREATE INDEX idx_oauth_connections_user_id ON oauth_connections(user_id);
CREATE INDEX idx_oauth_connections_provider ON oauth_connections(provider);
CREATE INDEX idx_oauth_connections_provider_id ON oauth_connections(provider, provider_id);
