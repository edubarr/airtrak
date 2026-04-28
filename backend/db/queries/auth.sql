-- name: CreateUser :one
INSERT INTO users (display_name, locale)
VALUES (@display_name, @locale)
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE id = @id
  AND deleted_at IS NULL;

-- name: CreateCredential :one
INSERT INTO user_credentials (user_id, type, identifier, secret_hash, verified_at)
VALUES (@user_id, @credential_type, @identifier, @secret_hash, @verified_at)
RETURNING *;

-- name: GetCredentialByTypeIdentifier :one
SELECT *
FROM user_credentials
WHERE type = @credential_type
  AND identifier = @identifier
  AND deleted_at IS NULL;

-- name: FindUserByCredential :one
SELECT users.*
FROM users
JOIN user_credentials ON user_credentials.user_id = users.id
WHERE user_credentials.type = @credential_type
  AND user_credentials.identifier = @identifier
  AND user_credentials.deleted_at IS NULL
  AND users.deleted_at IS NULL;

-- name: CreateSession :one
INSERT INTO user_sessions (user_id, expires_at)
VALUES (@user_id, @expires_at)
RETURNING *;

-- name: GetActiveSession :one
SELECT *
FROM user_sessions
WHERE id = @id
  AND revoked_at IS NULL
  AND expires_at > now();

-- name: RevokeSession :execrows
UPDATE user_sessions
SET revoked_at = now(), updated_at = now()
WHERE id = @id AND revoked_at IS NULL;
