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

-- name: GetAttemptForUpdate :one
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
  AND user_id = sqlc.arg(user_id)
FOR UPDATE;

-- name: ListAttemptsByCertification :many
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
WHERE user_id = sqlc.arg(user_id)
  AND certification_id = sqlc.arg(certification_id)
ORDER BY started_at DESC;

-- name: CreateAttempt :one
INSERT INTO attempts (user_id, certification_id, topic_id, mode, question_count, only_mistakes)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(certification_id),
    sqlc.narg(topic_id),
    sqlc.arg(mode),
    sqlc.arg(question_count),
    sqlc.arg(only_mistakes)
)
RETURNING
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
    updated_at;

-- name: CompleteAttempt :one
UPDATE attempts
SET score = sqlc.arg(score),
    correct_count = sqlc.arg(correct_count),
    completed_at = now(),
    duration_seconds = sqlc.narg(duration_seconds)
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
  AND completed_at IS NULL
RETURNING
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
    updated_at;

-- name: InsertAttemptQuestion :exec
INSERT INTO attempt_questions (attempt_id, question_id, position)
VALUES (sqlc.arg(attempt_id), sqlc.arg(question_id), sqlc.arg(position));

-- name: InsertAttemptAnswer :exec
INSERT INTO attempt_answers (attempt_id, question_id, is_correct)
VALUES (sqlc.arg(attempt_id), sqlc.arg(question_id), sqlc.arg(is_correct));

-- name: InsertAttemptAnswerOption :exec
INSERT INTO attempt_answer_options (attempt_id, question_id, option_id)
VALUES (sqlc.arg(attempt_id), sqlc.arg(question_id), sqlc.arg(option_id));

-- name: CountCompletedExams :one
SELECT COUNT(*)::int AS exam_count
FROM attempts
WHERE user_id = sqlc.arg(user_id)
  AND certification_id = sqlc.arg(certification_id)
  AND topic_id IS NULL
  AND completed_at IS NOT NULL;
