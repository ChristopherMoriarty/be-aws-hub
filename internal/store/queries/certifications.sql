-- name: ListCertifications :many
SELECT id, code, title, question_count, duration_seconds, passing_percent, created_at, updated_at
FROM certifications
ORDER BY code;

-- name: GetCertification :one
SELECT id, code, title, question_count, duration_seconds, passing_percent, created_at, updated_at
FROM certifications
WHERE id = sqlc.arg(id);
