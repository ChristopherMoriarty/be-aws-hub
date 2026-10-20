package service

import "testing"

func TestSelectWeightedSubtractsChosenWeight(t *testing.T) {
	calls := 0
	next := func() float64 {
		calls++
		if calls == 1 {
			return 0
		}
		return 0.4
	}
	got := selectWeighted([]float64{10, 1, 1}, 2, next)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("indexes = %v", got)
	}
}

func TestSelectWeightedKeepsPoolOrderWhenNothingIsLeftOut(t *testing.T) {
	got := selectWeighted([]float64{1, 5}, 2, func() float64 {
		t.Fatal("random source should not be used")
		return 0
	})
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("indexes = %v", got)
	}
}

func TestSelectWeightedPrefersHeavierQuestion(t *testing.T) {
	got := selectWeighted([]float64{1, 100}, 1, func() float64 { return 0.99 })
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("indexes = %v", got)
	}
}
