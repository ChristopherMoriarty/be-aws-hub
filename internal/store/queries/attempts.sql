-- name: GetAttempt :one
SELECT
    id,
    user_id,
    certification_id,
    topic_id,
    mode,
    question_count,
    only_mistakes,
    score,
    correct_count,
    started_at,
    completed_at,
    duration_seconds,
    created_at,
    updated_at
FROM attempts
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id);

-- name: ListAttemptQuestions :many
SELECT attempt_id, question_id, position
FROM attempt_questions
WHERE attempt_id = sqlc.arg(attempt_id)
ORDER BY position;

-- name: ListAttemptAnswers :many
SELECT attempt_id, question_id, is_correct, answered_at
FROM attempt_answers
WHERE attempt_id = sqlc.arg(attempt_id)
ORDER BY answered_at;

-- name: ListAttemptAnswerOptions :many
SELECT attempt_id, question_id, option_id
FROM attempt_answer_options
WHERE attempt_id = sqlc.arg(attempt_id);
