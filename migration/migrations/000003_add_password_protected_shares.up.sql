-- SQLite cannot drop a column in place, so add the password verifier and
-- remove the obsolete server-visible crypto version by rebuilding the table.
-- Both blobs are opaque ciphertext generated in the browser.
CREATE TABLE shares_new (
    id                BLOB PRIMARY KEY NOT NULL CHECK (length(id) = 32),
    encrypted_blob    BLOB NOT NULL CHECK (length(encrypted_blob) > 0),
    access_envelope   BLOB,
    created_at        INTEGER NOT NULL,
    expires_at        INTEGER NOT NULL,
    views_left        INTEGER CHECK (views_left IS NULL OR views_left >= 0),
    revoke_token_hash BLOB NOT NULL CHECK (length(revoke_token_hash) = 32),
    size_bytes        INTEGER NOT NULL CHECK (size_bytes > 0),
    CHECK (expires_at > created_at)
);

INSERT INTO shares_new (
    id, encrypted_blob, created_at, expires_at,
    views_left, revoke_token_hash, size_bytes
)
SELECT
    id, encrypted_blob, created_at, expires_at,
    views_left, revoke_token_hash, size_bytes
FROM shares;

DROP TABLE shares;
ALTER TABLE shares_new RENAME TO shares;

CREATE INDEX shares_expiry_idx ON shares (expires_at);
