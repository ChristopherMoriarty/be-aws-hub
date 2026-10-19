-- name: ListTopicsByCertification :many
SELECT id, certification_id, title, sort_order, created_at, updated_at
FROM topics
WHERE certification_id = sqlc.arg(certification_id)
ORDER BY sort_order, title;
