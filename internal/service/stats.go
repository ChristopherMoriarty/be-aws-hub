package service

import "math"

const (
	learningScoreFloor     = -1
	initialCorrectWeight   = 0.7
	initialWrongWeight     = 1.5
	weightIncreaseFactor   = 1.5
	weightDecreaseFactor   = 0.7
	weightFloor            = 0.2
	weightCeil             = 5.0
	lowAccuracy            = 0.5
	highAccuracy           = 0.8
	topicMasteredRatio     = 0.95
	topicTestQuestionCount = 10
)

type answerStat struct {
	TotalAttempts      int
	CorrectAttempts    int
	ConsecutiveCorrect int
	LearningScore      int
	Weight             float64
	LastCorrect        bool
	Exists             bool
}

func applyAnswer(prev answerStat, correct bool) answerStat {
	if !prev.Exists {
		next := answerStat{
			TotalAttempts: 1,
			LearningScore: learningScoreFloor,
			Weight:        initialWrongWeight,
			Exists:        true,
		}
		if correct {
			next.CorrectAttempts = 1
			next.ConsecutiveCorrect = 1
			next.LearningScore = 1
			next.Weight = initialCorrectWeight
			next.LastCorrect = true
		}
		return next
	}

	total := prev.TotalAttempts + 1
	correctCount := prev.CorrectAttempts
	consecutive := 0
	score := prev.LearningScore - 1
	if correct {
		correctCount++
		score = prev.LearningScore + 1
		if prev.LastCorrect {
			consecutive = prev.ConsecutiveCorrect + 1
		} else {
			consecutive = 1
		}
	}
	if score < learningScoreFloor {
		score = learningScoreFloor
	}

	accuracy := float64(correctCount) / float64(total)
	weight := prev.Weight
	if accuracy < lowAccuracy {
		weight = math.Min(weight*weightIncreaseFactor, weightCeil)
	} else if accuracy > highAccuracy && consecutive >= 2 {
		weight = math.Max(weight*weightDecreaseFactor, weightFloor)
	}

	return answerStat{
		TotalAttempts:      total,
		CorrectAttempts:    correctCount,
		ConsecutiveCorrect: consecutive,
		LearningScore:      score,
		Weight:             weight,
		LastCorrect:        correct,
		Exists:             true,
	}
}

func topicMastered(total, attempted, mastered int) bool {
	if total <= 0 || attempted <= 0 {
		return false
	}
	return float64(attempted)/float64(total) >= topicMasteredRatio && mastered == attempted
}

func answerAccuracy(correct, total int) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(correct)/float64(total)*1000) / 10
}

func percentScore(correct, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(correct) / float64(total) * 100))
}
