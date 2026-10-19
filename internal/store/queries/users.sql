-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users
WHERE lower(email) = lower(sqlc.arg(email));

-- name: GetUserByID :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users
WHERE id = sqlc.arg(id);
