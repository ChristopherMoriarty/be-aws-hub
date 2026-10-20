package service

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type InvalidError struct {
	Message string
}

func (e *InvalidError) Error() string {
	return e.Message
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

func invalid(message string) error {
	return &InvalidError{Message: message}
}

func conflict(message string) error {
	return &ConflictError{Message: message}
}

func notFound(message string) error {
	return &NotFoundError{Message: message}
}

func storeErr(op string, err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return conflict(uniqueMessage(pgErr.ConstraintName))
		case "23503":
			if message, ok := missingReference(pgErr.ConstraintName); ok {
				return notFound(message)
			}
			return conflict(referenceMessage(pgErr.ConstraintName))
		case "23514":
			return invalid("invalid value")
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

func uniqueMessage(constraint string) string {
	switch constraint {
	case "certifications_code_unique":
		return "certification code already exists"
	case "topics_certification_title_unique":
		return "topic title already exists"
	default:
		return "conflict"
	}
}

func missingReference(constraint string) (string, bool) {
	switch constraint {
	case "topics_certification_id_fkey":
		return "certification not found", true
	case "questions_topic_id_fkey":
		return "topic not found", true
	default:
		return "", false
	}
}

func referenceMessage(constraint string) string {
	switch constraint {
	case "attempts_certification_id_fkey":
		return "certification is used by attempts"
	case "attempts_topic_certification_fkey":
		return "topic is used by attempts"
	case "attempt_questions_question_id_fkey":
		return "question is used by attempts"
	case "attempt_answer_options_option_question_fkey":
		return "question options are used by attempts"
	default:
		return "resource is in use"
	}
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func readErr(err error, missing, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(missing)
	}
	return fmt.Errorf("%s: %w", op, err)
}
