package tutor

import (
	"errors"
	"testing"
)

func TestTransition(t *testing.T) {
	cases := []struct {
		phase    Phase
		turns    int
		action   Action
		want     Phase
		advanced bool
	}{
		{PhaseIntroduction, 1, ActionAdvancePhase, PhaseExplanation, true},
		{PhaseIntroduction, 1, ActionContinue, PhaseIntroduction, false},
		{PhaseGuidedPractice, 1, ActionAdvancePhase, PhaseGuidedPractice, false}, // below minTurns
		{PhaseGuidedPractice, 2, ActionAdvancePhase, PhaseIndependentPractice, true},
		{PhaseGuidedPractice, 8, ActionAskRetry, PhaseIndependentPractice, true}, // maxTurns safety valve
		{PhaseIndependentPractice, 2, ActionAdvancePhase, PhaseAssessment, true},
		{PhaseReview, 2, ActionAdvancePhase, PhaseAssessment, true},
		{PhaseAssessment, 5, ActionAdvancePhase, PhaseAssessment, false},
		{PhaseMastered, 1, ActionAdvancePhase, PhaseMastered, false},
	}
	for _, c := range cases {
		got, adv := Transition(c.phase, c.turns, c.action)
		if got != c.want || adv != c.advanced {
			t.Errorf("Transition(%s,%d,%s) = %s,%v; want %s,%v", c.phase, c.turns, c.action, got, adv, c.want, c.advanced)
		}
	}
}

func TestParseReply(t *testing.T) {
	r, err := ParseReply([]byte(`{"message":"  Try again! ","action":"ask_retry","observedMistakes":["verb-form","BAD TAG!!",""]}`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Message != "Try again!" || r.Action != ActionAskRetry || len(r.ObservedMistakes) != 1 || r.ObservedMistakes[0] != "verb-form" {
		t.Fatalf("unexpected reply %+v", r)
	}
	for _, bad := range []string{
		`not json`,
		`{"message":"","action":"continue","observedMistakes":[]}`,
		`{"message":"hi","action":"mark_mastered","observedMistakes":[]}`,
	} {
		if _, err := ParseReply([]byte(bad)); !errors.Is(err, ErrMalformed) {
			t.Errorf("ParseReply(%q) err = %v, want ErrMalformed", bad, err)
		}
	}
}
