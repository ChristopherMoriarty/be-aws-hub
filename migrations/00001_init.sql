-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_not_blank CHECK (btrim(email) <> ''),
    CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'))
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

CREATE TABLE certifications (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code             TEXT NOT NULL,
    title            TEXT NOT NULL,
    question_count   INT NOT NULL,
    duration_seconds INT NOT NULL,
    passing_percent  INT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT certifications_code_unique UNIQUE (code),
    CONSTRAINT certifications_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT certifications_title_not_blank CHECK (btrim(title) <> ''),
    CONSTRAINT certifications_question_count_check CHECK (question_count > 0),
    CONSTRAINT certifications_duration_check CHECK (duration_seconds > 0),
    CONSTRAINT certifications_passing_percent_check CHECK (passing_percent BETWEEN 0 AND 100)
);

CREATE TABLE topics (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    certification_id UUID NOT NULL,
    title            TEXT NOT NULL,
    sort_order       INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT topics_certification_id_fkey
        FOREIGN KEY (certification_id) REFERENCES certifications (id) ON DELETE CASCADE,
    CONSTRAINT topics_title_not_blank CHECK (btrim(title) <> ''),
    CONSTRAINT topics_certification_title_unique UNIQUE (certification_id, title),
    -- Pair key so an attempt can require the topic to belong to its certification.
    CONSTRAINT topics_id_certification_unique UNIQUE (id, certification_id)
);

CREATE INDEX topics_certification_sort_idx ON topics (certification_id, sort_order);

CREATE TABLE questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id    UUID NOT NULL,
    body        TEXT NOT NULL,
    explanation TEXT,
    difficulty  TEXT NOT NULL DEFAULT 'medium',
    image_url   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT questions_topic_id_fkey
        FOREIGN KEY (topic_id) REFERENCES topics (id) ON DELETE CASCADE,
    CONSTRAINT questions_body_not_blank CHECK (btrim(body) <> ''),
    CONSTRAINT questions_difficulty_check CHECK (difficulty IN ('easy', 'medium', 'hard')),
    -- Pair key so an option row can be tied to this question, not only to an option id.
    CONSTRAINT questions_id_topic_unique UNIQUE (id, topic_id)
);

CREATE INDEX questions_topic_id_idx ON questions (topic_id);

CREATE TABLE question_options (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL,
    body        TEXT NOT NULL,
    is_correct  BOOLEAN NOT NULL DEFAULT false,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT question_options_question_id_fkey
        FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE,
    CONSTRAINT question_options_body_not_blank CHECK (btrim(body) <> ''),
    CONSTRAINT question_options_question_sort_unique
        UNIQUE (question_id, sort_order) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT question_options_id_question_unique UNIQUE (id, question_id)
);

CREATE TABLE attempts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    certification_id UUID NOT NULL,
    topic_id         UUID,
    mode             TEXT NOT NULL,
    question_count   INT NOT NULL,
    only_mistakes    BOOLEAN NOT NULL DEFAULT false,
    score            INT,
    correct_count    INT,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at     TIMESTAMPTZ,
    duration_seconds INT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT attempts_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT attempts_certification_id_fkey
        FOREIGN KEY (certification_id) REFERENCES certifications (id) ON DELETE RESTRICT,
    -- NULL topic_id skips this check. A set topic_id must belong to certification_id.
    CONSTRAINT attempts_topic_certification_fkey
        FOREIGN KEY (topic_id, certification_id)
        REFERENCES topics (id, certification_id) ON DELETE RESTRICT,
    CONSTRAINT attempts_mode_check CHECK (mode IN ('exam', 'learning')),
    CONSTRAINT attempts_question_count_check CHECK (question_count > 0),
    CONSTRAINT attempts_score_check CHECK (score IS NULL OR score BETWEEN 0 AND 100),
    CONSTRAINT attempts_correct_count_check CHECK (
        correct_count IS NULL OR (correct_count >= 0 AND correct_count <= question_count)
    ),
    CONSTRAINT attempts_result_check CHECK (
        (completed_at IS NULL AND score IS NULL AND correct_count IS NULL)
        OR (completed_at IS NOT NULL AND score IS NOT NULL AND correct_count IS NOT NULL)
    ),
    CONSTRAINT attempts_duration_check CHECK (
        duration_seconds IS NULL
        OR (duration_seconds >= 0 AND completed_at IS NOT NULL)
    )
);

CREATE INDEX attempts_user_cert_started_idx
    ON attempts (user_id, certification_id, started_at DESC);

CREATE INDEX attempts_topic_id_idx
    ON attempts (topic_id)
    WHERE topic_id IS NOT NULL;

CREATE TABLE attempt_questions (
    attempt_id  UUID NOT NULL,
    question_id UUID NOT NULL,
    position    INT NOT NULL,
    CONSTRAINT attempt_questions_pkey PRIMARY KEY (attempt_id, question_id),
    CONSTRAINT attempt_questions_attempt_id_fkey
        FOREIGN KEY (attempt_id) REFERENCES attempts (id) ON DELETE CASCADE,
    CONSTRAINT attempt_questions_question_id_fkey
        FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE RESTRICT,
    CONSTRAINT attempt_questions_position_unique UNIQUE (attempt_id, position),
    CONSTRAINT attempt_questions_position_check CHECK (position > 0)
);

CREATE INDEX attempt_questions_question_id_idx ON attempt_questions (question_id);

CREATE TABLE attempt_answers (
    attempt_id  UUID NOT NULL,
    question_id UUID NOT NULL,
    is_correct  BOOLEAN NOT NULL,
    answered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT attempt_answers_pkey PRIMARY KEY (attempt_id, question_id),
    CONSTRAINT attempt_answers_attempt_question_fkey
        FOREIGN KEY (attempt_id, question_id)
        REFERENCES attempt_questions (attempt_id, question_id)
        ON DELETE CASCADE
);

CREATE TABLE attempt_answer_options (
    attempt_id  UUID NOT NULL,
    question_id UUID NOT NULL,
    option_id   UUID NOT NULL,
    CONSTRAINT attempt_answer_options_pkey PRIMARY KEY (attempt_id, question_id, option_id),
    CONSTRAINT attempt_answer_options_answer_fkey
        FOREIGN KEY (attempt_id, question_id)
        REFERENCES attempt_answers (attempt_id, question_id)
        ON DELETE CASCADE,
    CONSTRAINT attempt_answer_options_option_question_fkey
        FOREIGN KEY (option_id, question_id)
        REFERENCES question_options (id, question_id)
        ON DELETE RESTRICT
);

CREATE INDEX attempt_answer_options_option_id_idx
    ON attempt_answer_options (option_id);

CREATE TABLE question_stats (
    user_id             UUID NOT NULL,
    question_id         UUID NOT NULL,
    total_attempts      INT NOT NULL DEFAULT 0,
    correct_attempts    INT NOT NULL DEFAULT 0,
    consecutive_correct INT NOT NULL DEFAULT 0,
    learning_score      INT NOT NULL DEFAULT 0,
    weight              DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    last_attempted_at   TIMESTAMPTZ,
    last_correct        BOOLEAN,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT question_stats_pkey PRIMARY KEY (user_id, question_id),
    CONSTRAINT question_stats_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT question_stats_question_id_fkey
        FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE CASCADE,
    CONSTRAINT question_stats_counts_check CHECK (
        total_attempts >= 0
        AND correct_attempts >= 0
        AND correct_attempts <= total_attempts
        AND consecutive_correct >= 0
        AND learning_score >= -1
    ),
    CONSTRAINT question_stats_weight_check CHECK (weight > 0 AND weight <= 5)
);

CREATE INDEX question_stats_question_id_idx ON question_stats (question_id);

CREATE INDEX question_stats_user_learning_idx
    ON question_stats (user_id, learning_score);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER certifications_set_updated_at
    BEFORE UPDATE ON certifications
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER topics_set_updated_at
    BEFORE UPDATE ON topics
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER questions_set_updated_at
    BEFORE UPDATE ON questions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER question_options_set_updated_at
    BEFORE UPDATE ON question_options
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER attempts_set_updated_at
    BEFORE UPDATE ON attempts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER question_stats_set_updated_at
    BEFORE UPDATE ON question_stats
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE question_stats;
DROP TABLE attempt_answer_options;
DROP TABLE attempt_answers;
DROP TABLE attempt_questions;
DROP TABLE attempts;
DROP TABLE question_options;
DROP TABLE questions;
DROP TABLE topics;
DROP TABLE certifications;
DROP TABLE users;
DROP FUNCTION set_updated_at();
