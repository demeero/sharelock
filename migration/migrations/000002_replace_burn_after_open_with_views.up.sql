-- SQLite cannot add a CHECK constraint with ALTER TABLE, and it refuses to drop
-- a column that a CHECK constraint references, so burn_after_open can only be
-- replaced by rebuilding the table. golang-migrate wraps this file in a
-- transaction and the schema has no foreign keys, so the rebuild is atomic and
-- needs no foreign_keys pragma dance.
CREATE TABLE shares_new (
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

INSERT INTO shares_new (
    id, encrypted_blob, crypto_version, created_at, expires_at,
    views_left, revoke_token_hash, size_bytes
)
SELECT
    id, encrypted_blob, crypto_version, created_at, expires_at,
    CASE WHEN burn_after_open = 1 THEN 1 ELSE NULL END,
    revoke_token_hash, size_bytes
FROM shares;

-- Dropping the table drops shares_expiry_idx with it, so recreate the index.
DROP TABLE shares;
ALTER TABLE shares_new RENAME TO shares;

CREATE INDEX shares_expiry_idx ON shares (expires_at);
