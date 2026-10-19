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
