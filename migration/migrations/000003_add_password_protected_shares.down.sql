-- The former schema cannot represent an access verifier. The INSERT deliberately
-- fails when one exists, so golang-migrate rolls this transaction back instead
-- of silently discarding password-protected shares.
CREATE TABLE shares_old (
    id                BLOB PRIMARY KEY NOT NULL CHECK (length(id) = 32),
    encrypted_blob    BLOB NOT NULL CHECK (length(encrypted_blob) > 0),
    crypto_version    INTEGER NOT NULL CHECK (crypto_version = 1),
    created_at        INTEGER NOT NULL,
    expires_at        INTEGER NOT NULL,
    views_left        INTEGER CHECK (views_left IS NULL OR views_left >= 0),
    revoke_token_hash BLOB NOT NULL CHECK (length(revoke_token_hash) = 32),
    size_bytes        INTEGER NOT NULL CHECK (size_bytes > 0),
    CHECK (expires_at > created_at)
);

INSERT INTO shares_old (
    id, encrypted_blob, crypto_version, created_at, expires_at,
    views_left, revoke_token_hash, size_bytes
)
SELECT
    id, encrypted_blob,
    CASE WHEN access_envelope IS NULL THEN 1 ELSE 2 END,
    created_at, expires_at,
    views_left, revoke_token_hash, size_bytes
FROM shares;

DROP TABLE shares;
ALTER TABLE shares_old RENAME TO shares;

CREATE INDEX shares_expiry_idx ON shares (expires_at);
