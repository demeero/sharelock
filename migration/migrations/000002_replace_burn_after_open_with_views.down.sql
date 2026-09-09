-- Mirror rebuild back to the boolean column. This is lossy: a share with more
-- than one remaining view collapses into burn_after_open = 1, because the old
-- schema cannot express a counter.
CREATE TABLE shares_old (
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

INSERT INTO shares_old (
    id, encrypted_blob, crypto_version, created_at, expires_at,
    burn_after_open, revoke_token_hash, size_bytes
)
SELECT
    id, encrypted_blob, crypto_version, created_at, expires_at,
    CASE WHEN views_left IS NULL THEN 0 ELSE 1 END,
    revoke_token_hash, size_bytes
FROM shares;

DROP TABLE shares;
ALTER TABLE shares_old RENAME TO shares;

CREATE INDEX shares_expiry_idx ON shares (expires_at);
