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
