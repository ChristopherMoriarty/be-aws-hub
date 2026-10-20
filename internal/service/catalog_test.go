package service

import "testing"

func TestCertificationParamsRejectsExamSize(t *testing.T) {
	_, err := certificationParams(CertificationInput{
		Code:            " saa-c03 ",
		Title:           "Solutions Architect",
		QuestionCount:   0,
		DurationSeconds: 130 * 60,
		PassingPercent:  72,
	})
	if err == nil || err.Error() != "question_count must be positive" {
		t.Fatalf("error = %v", err)
	}
}
