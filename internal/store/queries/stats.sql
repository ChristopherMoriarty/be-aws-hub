-- name: GetQuestionStat :one
SELECT
    user_id,
    question_id,
    total_attempts,
    correct_attempts,
    consecutive_correct,
    learning_score,
    weight,
    last_attempted_at,
    last_correct,
    created_at,
    updated_at
FROM question_stats
WHERE user_id = sqlc.arg(user_id)
  AND question_id = sqlc.arg(question_id);

-- name: GetQuestionStatForUpdate :one
SELECT
    user_id,
    question_id,
    total_attempts,
    correct_attempts,
    consecutive_correct,
    learning_score,
    weight,
    last_attempted_at,
    last_correct,
    created_at,
    updated_at
FROM question_stats
WHERE user_id = sqlc.arg(user_id)
  AND question_id = sqlc.arg(question_id)
FOR UPDATE;

-- name: InsertQuestionStat :exec
INSERT INTO question_stats (
    user_id,
    question_id,
    total_attempts,
    correct_attempts,
    consecutive_correct,
    learning_score,
    weight,
    last_attempted_at,
    last_correct
) VALUES (
    sqlc.arg(user_id),
    sqlc.arg(question_id),
    sqlc.arg(total_attempts),
    sqlc.arg(correct_attempts),
    sqlc.arg(consecutive_correct),
    sqlc.arg(learning_score),
    sqlc.arg(weight),
    now(),
    sqlc.arg(last_correct)
);

-- name: UpdateQuestionStat :exec
UPDATE question_stats
SET total_attempts = sqlc.arg(total_attempts),
    correct_attempts = sqlc.arg(correct_attempts),
    consecutive_correct = sqlc.arg(consecutive_correct),
    learning_score = sqlc.arg(learning_score),
    weight = sqlc.arg(weight),
    last_attempted_at = now(),
    last_correct = sqlc.arg(last_correct)
WHERE user_id = sqlc.arg(user_id)
  AND question_id = sqlc.arg(question_id);

-- name: GetCertificationStats :one
SELECT
    COUNT(q.id)::int AS total_questions,
    COUNT(q.id) FILTER (WHERE s.total_attempts > 0)::int AS attempted_questions,
    COUNT(q.id) FILTER (WHERE s.total_attempts > 0 AND s.learning_score >= 1)::int AS mastered_questions,
    COUNT(q.id) FILTER (WHERE s.total_attempts > 0 AND s.learning_score < 1)::int AS needs_review,
    COALESCE(SUM(s.total_attempts), 0)::int AS total_attempts,
    COALESCE(SUM(s.correct_attempts), 0)::int AS correct_answers
FROM questions q
JOIN topics t ON t.id = q.topic_id
LEFT JOIN question_stats s
    ON s.question_id = q.id
   AND s.user_id = sqlc.arg(user_id)
WHERE t.certification_id = sqlc.arg(certification_id);

-- name: ListTopicProgress :many
SELECT
    t.id,
    t.title,
    t.sort_order,
    COUNT(q.id)::int AS total_questions,
    COUNT(q.id) FILTER (WHERE s.total_attempts > 0)::int AS attempted_questions,
    COUNT(q.id) FILTER (WHERE s.total_attempts > 0 AND s.learning_score >= 1)::int AS mastered_questions,
    COALESCE(SUM(s.total_attempts), 0)::int AS total_attempts,
    COALESCE(SUM(s.correct_attempts), 0)::int AS correct_answers
FROM topics t
LEFT JOIN questions q ON q.topic_id = t.id
LEFT JOIN question_stats s
    ON s.question_id = q.id
   AND s.user_id = sqlc.arg(user_id)
WHERE t.certification_id = sqlc.arg(certification_id)
GROUP BY t.id, t.title, t.sort_order
ORDER BY t.sort_order, t.title;
