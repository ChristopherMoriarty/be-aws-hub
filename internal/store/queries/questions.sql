-- name: ListQuestionsByTopic :many
SELECT id, topic_id, body, explanation, difficulty, image_url, created_at, updated_at
FROM questions
WHERE topic_id = sqlc.arg(topic_id)
ORDER BY created_at, id;

-- name: ListOptionsByQuestion :many
SELECT id, question_id, body, is_correct, sort_order, created_at, updated_at
FROM question_options
WHERE question_id = sqlc.arg(question_id)
ORDER BY sort_order;

-- name: ListQuestionsForAttempt :many
SELECT
    q.id,
    q.topic_id,
    q.body,
    q.explanation,
    q.difficulty,
    q.image_url,
    q.created_at,
    q.updated_at,
    COALESCE(s.weight, 1.0)::float8 AS weight
FROM questions q
JOIN topics t ON t.id = q.topic_id
LEFT JOIN question_stats s
    ON s.question_id = q.id
   AND s.user_id = sqlc.arg(user_id)
WHERE t.certification_id = sqlc.arg(certification_id)
  AND (
    sqlc.narg(topic_id)::uuid IS NULL
    OR q.topic_id = sqlc.narg(topic_id)
  )
  AND (
    sqlc.arg(only_mistakes)::bool = false
    OR s.user_id IS NULL
    OR s.total_attempts = 0
    OR s.learning_score < 1
  )
ORDER BY COALESCE(s.learning_score, 0), COALESCE(s.weight, 1.0) DESC, q.id;

-- name: ListQuestionsByIDs :many
SELECT id, topic_id, body, explanation, difficulty, image_url, created_at, updated_at
FROM questions
WHERE id = ANY(sqlc.arg(question_ids)::uuid[]);

-- name: ListOptionsByQuestionIDs :many
SELECT id, question_id, body, is_correct, sort_order, created_at, updated_at
FROM question_options
WHERE question_id = ANY(sqlc.arg(question_ids)::uuid[])
ORDER BY sort_order, id;

-- name: GetQuestion :one
SELECT id, topic_id, body, explanation, difficulty, image_url, created_at, updated_at
FROM questions
WHERE id = sqlc.arg(id);

-- name: CreateQuestion :one
INSERT INTO questions (topic_id, body, explanation, difficulty, image_url)
VALUES (
    sqlc.arg(topic_id),
    sqlc.arg(body),
    sqlc.arg(explanation),
    sqlc.arg(difficulty),
    sqlc.arg(image_url)
)
RETURNING id, topic_id, body, explanation, difficulty, image_url, created_at, updated_at;

-- name: UpdateQuestion :one
UPDATE questions
SET body = sqlc.arg(body),
    explanation = sqlc.arg(explanation),
    difficulty = sqlc.arg(difficulty),
    image_url = sqlc.arg(image_url)
WHERE id = sqlc.arg(id)
RETURNING id, topic_id, body, explanation, difficulty, image_url, created_at, updated_at;

-- name: DeleteQuestion :execresult
DELETE FROM questions
WHERE id = sqlc.arg(id);

-- name: CreateQuestionOption :one
INSERT INTO question_options (question_id, body, is_correct, sort_order)
VALUES (
    sqlc.arg(question_id),
    sqlc.arg(body),
    sqlc.arg(is_correct),
    sqlc.arg(sort_order)
)
RETURNING id, question_id, body, is_correct, sort_order, created_at, updated_at;

-- name: DeleteOptionsByQuestion :exec
DELETE FROM question_options
WHERE question_id = sqlc.arg(question_id);
