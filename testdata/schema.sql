CREATE TABLE "shares" (
    id                BLOB PRIMARY KEY NOT NULL CHECK (length(id) = 32),
    encrypted_blob    BLOB NOT NULL CHECK (length(encrypted_blob) > 0),
    crypto_version    INTEGER NOT NULL CHECK (crypto_version = 1),
    created_at        INTEGER NOT NULL,
    expires_at        INTEGER NOT NULL,
    -- NULL means unlimited opens. Zero is reachable only inside the open
    -- transaction, between the decrement and the delete of the last view.
    views_left        INTEGER CHECK (views_left IS NULL OR views_left >= 0),
    revoke_token_hash BLOB NOT NULL CHECK (length(revoke_token_hash) = 32),
    size_bytes        INTEGER NOT NULL CHECK (size_bytes > 0),
    CHECK (expires_at > created_at)
);
CREATE INDEX shares_expiry_idx ON shares (expires_at);
