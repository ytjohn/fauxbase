-- name: GetActiveSigningKey :one
SELECT * FROM signing_keys 
WHERE is_active = 1 
AND expires_at > CURRENT_TIMESTAMP 
ORDER BY expires_at DESC 
LIMIT 1;

-- name: CreateSigningKey :one
INSERT INTO signing_keys (key_data, expires_at, is_active) 
VALUES (?, datetime('now', '+5 days'), 1)
RETURNING *;

-- name: UpdateSigningKeyData :exec
UPDATE signing_keys 
SET key_data = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteExpiredKeys :exec
DELETE FROM signing_keys 
WHERE expires_at < datetime('now', @days || ' days');

-- name: GetAllValidSigningKeys :many
SELECT * FROM signing_keys 
WHERE is_active = 1 
AND expires_at > CURRENT_TIMESTAMP 
ORDER BY expires_at DESC;

-- name: MarkExpiredKeysInactive :exec
UPDATE signing_keys 
SET is_active = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE expires_at < CURRENT_TIMESTAMP 
AND is_active = 1;
