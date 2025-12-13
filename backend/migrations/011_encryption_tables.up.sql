-- User Keys Table
CREATE TABLE IF NOT EXISTS user_keys (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Process Keys Table
CREATE TABLE IF NOT EXISTS process_keys (
    process_id VARCHAR(100) PRIMARY KEY,
    encrypted_key_material TEXT NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_process_keys_expires_at ON process_keys(expires_at);

-- Encrypted Records Table
CREATE TABLE IF NOT EXISTS encrypted_records (
    id BIGSERIAL PRIMARY KEY,
    process_id VARCHAR(100) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Encrypted data
    encrypted_data BYTEA NOT NULL,
    encryption_nonce BYTEA NOT NULL,
    encryption_tag BYTEA NOT NULL,
    
    -- Key metadata
    key_version INT NOT NULL DEFAULT 1,
    system_key_id VARCHAR(50) NOT NULL,
    process_key_hash VARCHAR(64) NOT NULL,
    
    -- Digital signature
    data_signature TEXT NOT NULL,
    signature_public_key TEXT NOT NULL,
    
    -- Metadata (unencrypted for queries)
    record_type VARCHAR(50),
    status VARCHAR(50),
    
    -- Hash chain for audit trail
    prev_record_hash VARCHAR(64),
    chain_hash VARCHAR(64) NOT NULL UNIQUE,
    
    CONSTRAINT fk_process_key FOREIGN KEY (process_id) REFERENCES process_keys(process_id)
);

CREATE INDEX IF NOT EXISTS idx_encrypted_records_process_id ON encrypted_records(process_id);
CREATE INDEX IF NOT EXISTS idx_encrypted_records_user_id ON encrypted_records(user_id);
CREATE INDEX IF NOT EXISTS idx_encrypted_records_chain_hash ON encrypted_records(chain_hash);
CREATE INDEX IF NOT EXISTS idx_encrypted_records_record_type ON encrypted_records(record_type);

-- Prevent modifications (append-only)
CREATE OR REPLACE FUNCTION prevent_encrypted_record_modifications()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' OR TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'Modifications not allowed on encrypted_records - this is an append-only table';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS enforce_encrypted_records_append_only ON encrypted_records;
CREATE TRIGGER enforce_encrypted_records_append_only
    BEFORE UPDATE OR DELETE ON encrypted_records
    FOR EACH ROW EXECUTE FUNCTION prevent_encrypted_record_modifications();

-- Comments
COMMENT ON TABLE user_keys IS 'Stores user Ed25519 key pairs (private keys encrypted with Vault)';
COMMENT ON TABLE process_keys IS 'Stores process-specific symmetric encryption keys (encrypted with Vault)';
COMMENT ON TABLE encrypted_records IS 'Append-only table for encrypted and signed records with hash chain audit trail';
COMMENT ON COLUMN encrypted_records.encrypted_data IS 'AES-256-GCM encrypted payload';
COMMENT ON COLUMN encrypted_records.encryption_nonce IS 'GCM nonce/IV for decryption';
COMMENT ON COLUMN encrypted_records.encryption_tag IS 'GCM authentication tag';
COMMENT ON COLUMN encrypted_records.data_signature IS 'Ed25519 signature of encrypted data';
COMMENT ON COLUMN encrypted_records.chain_hash IS 'SHA-256 hash linking to previous record for tamper detection';
