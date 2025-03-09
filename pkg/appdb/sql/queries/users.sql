-- name: CreateUser :one
INSERT INTO users (
    username, email, password, role
) VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = ? LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = ? LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY username;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET lastlogin = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE users
SET password = ?, updated = CURRENT_TIMESTAMP
WHERE email = ?;

-- name: DeleteUser :exec
DELETE FROM users
WHERE email = ?;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = ? LIMIT 1; 