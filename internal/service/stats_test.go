package service

import "testing"

func TestApplyAnswerInitialWeights(t *testing.T) {
	correct := applyAnswer(answerStat{}, true)
	if correct.LearningScore != 1 || correct.Weight != initialCorrectWeight || correct.ConsecutiveCorrect != 1 {
		t.Fatalf("correct = %+v", correct)
	}
	wrong := applyAnswer(answerStat{}, false)
	if wrong.LearningScore != learningScoreFloor || wrong.Weight != initialWrongWeight || wrong.ConsecutiveCorrect != 0 {
		t.Fatalf("wrong = %+v", wrong)
	}
}

func TestApplyAnswerFloorsLearningScoreAndCapsWeight(t *testing.T) {
	next := applyAnswer(answerStat{
		Exists:             true,
		TotalAttempts:      1,
		LearningScore:      learningScoreFloor,
		Weight:             4,
		ConsecutiveCorrect: 0,
	}, false)
	if next.LearningScore != learningScoreFloor {
		t.Fatalf("score = %d", next.LearningScore)
	}
	if next.Weight != weightCeil {
		t.Fatalf("weight = %v", next.Weight)
	}
}

func TestApplyAnswerDecreasesWeightAfterAStreak(t *testing.T) {
	next := applyAnswer(answerStat{
		Exists:             true,
		TotalAttempts:      5,
		CorrectAttempts:    5,
		ConsecutiveCorrect: 2,
		LearningScore:      2,
		Weight:             1,
		LastCorrect:        true,
	}, true)
	if next.ConsecutiveCorrect != 3 || next.Weight != weightDecreaseFactor {
		t.Fatalf("next = %+v", next)
	}
}

func TestApplyAnswerKeepsWeightOnAccuracyBoundaries(t *testing.T) {
	half := applyAnswer(answerStat{
		Exists:          true,
		TotalAttempts:   1,
		CorrectAttempts: 1,
		LearningScore:   1,
		Weight:          2,
		LastCorrect:     true,
	}, false)
	if half.Weight != 2 {
		t.Fatalf("half weight = %v", half.Weight)
	}

	high := applyAnswer(answerStat{
		Exists:             true,
		TotalAttempts:      4,
		CorrectAttempts:    4,
		ConsecutiveCorrect: 4,
		LearningScore:      4,
		Weight:             2,
		LastCorrect:        true,
	}, false)
	if high.Weight != 2 {
		t.Fatalf("high boundary weight = %v", high.Weight)
	}
}

func TestTopicMasteredRequiresCoverageAndCleanAttempts(t *testing.T) {
	if !topicMastered(20, 19, 19) {
		t.Fatal("expected 95 percent attempted and all clean to be mastered")
	}
	if topicMastered(20, 18, 18) || topicMastered(20, 19, 18) || topicMastered(20, 0, 0) {
		t.Fatal("expected incomplete or mixed topics to stay open")
	}
}
