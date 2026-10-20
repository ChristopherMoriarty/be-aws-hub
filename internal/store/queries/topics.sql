-- name: ListTopicsByCertification :many
SELECT id, certification_id, title, sort_order, created_at, updated_at
FROM topics
WHERE certification_id = sqlc.arg(certification_id)
ORDER BY sort_order, title;

-- name: GetTopic :one
SELECT id, certification_id, title, sort_order, created_at, updated_at
FROM topics
WHERE id = sqlc.arg(id);

-- name: NextTopicSortOrder :one
SELECT COALESCE(MAX(sort_order) + 1, 0)::int AS sort_order
FROM topics
WHERE certification_id = sqlc.arg(certification_id);

-- name: CreateTopic :one
INSERT INTO topics (certification_id, title, sort_order)
VALUES (sqlc.arg(certification_id), sqlc.arg(title), sqlc.arg(sort_order))
RETURNING id, certification_id, title, sort_order, created_at, updated_at;

-- name: UpdateTopic :one
UPDATE topics
SET title = sqlc.arg(title),
    sort_order = sqlc.arg(sort_order)
WHERE id = sqlc.arg(id)
RETURNING id, certification_id, title, sort_order, created_at, updated_at;

-- name: DeleteTopic :execresult
DELETE FROM topics
WHERE id = sqlc.arg(id);
