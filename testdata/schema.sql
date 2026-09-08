CREATE TABLE shares (
    id                BLOB PRIMARY KEY NOT NULL CHECK (length(id) = 32),
    encrypted_blob    BLOB NOT NULL CHECK (length(encrypted_blob) > 0),
    crypto_version    INTEGER NOT NULL CHECK (crypto_version = 1),
    created_at        INTEGER NOT NULL,
    expires_at        INTEGER NOT NULL,
    burn_after_open   INTEGER NOT NULL CHECK (burn_after_open IN (0, 1)),
    revoke_token_hash BLOB NOT NULL CHECK (length(revoke_token_hash) = 32),
    size_bytes        INTEGER NOT NULL CHECK (size_bytes > 0),
    CHECK (expires_at > created_at)
);
CREATE INDEX shares_expiry_idx ON shares (expires_at);
