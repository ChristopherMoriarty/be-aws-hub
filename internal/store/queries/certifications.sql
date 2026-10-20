-- name: ListCertifications :many
SELECT id, code, title, question_count, duration_seconds, passing_percent, created_at, updated_at
FROM certifications
ORDER BY code;

-- name: GetCertification :one
SELECT id, code, title, question_count, duration_seconds, passing_percent, created_at, updated_at
FROM certifications
WHERE id = sqlc.arg(id);

-- name: CreateCertification :one
INSERT INTO certifications (code, title, question_count, duration_seconds, passing_percent)
VALUES (
    sqlc.arg(code),
    sqlc.arg(title),
    sqlc.arg(question_count),
    sqlc.arg(duration_seconds),
    sqlc.arg(passing_percent)
)
RETURNING id, code, title, question_count, duration_seconds, passing_percent, created_at, updated_at;

-- name: UpdateCertification :one
UPDATE certifications
SET code = sqlc.arg(code),
    title = sqlc.arg(title),
    question_count = sqlc.arg(question_count),
    duration_seconds = sqlc.arg(duration_seconds),
    passing_percent = sqlc.arg(passing_percent)
WHERE id = sqlc.arg(id)
RETURNING id, code, title, question_count, duration_seconds, passing_percent, created_at, updated_at;

-- name: DeleteCertification :execresult
DELETE FROM certifications
WHERE id = sqlc.arg(id);
