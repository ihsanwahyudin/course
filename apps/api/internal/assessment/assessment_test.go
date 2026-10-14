package assessment

import (
	"errors"
	"testing"

	"fluentpath/api/internal/curriculum"
)

var node = &curriculum.Node{ID: "n", MasteryThreshold: 80, Objectives: []curriculum.Objective{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}}

func TestScorePass(t *testing.T) {
	r, err := Score([]byte(`{"objectiveScores":{"a":90,"b":82},"evidence":["ok"],"feedback":"Good."}`), node)
	if err != nil {
		t.Fatal(err)
	}
	if r.Score != 86 || !r.Passed || len(r.WeakObjectives) != 0 {
		t.Fatalf("got %+v", r)
	}
}

func TestScoreBelowThreshold(t *testing.T) {
	r, err := Score([]byte(`{"objectiveScores":{"a":100,"b":40},"evidence":[],"feedback":"Review b."}`), node)
	if err != nil {
		t.Fatal(err)
	}
	if r.Score != 70 || r.Passed || len(r.WeakObjectives) != 1 || r.WeakObjectives[0] != "b" {
		t.Fatalf("got %+v", r)
	}
}

func TestScoreBoundaryIsInclusive(t *testing.T) {
	r, err := Score([]byte(`{"objectiveScores":{"a":80,"b":80},"evidence":[],"feedback":"ok"}`), node)
	if err != nil || !r.Passed {
		t.Fatalf("80 should pass: %+v %v", r, err)
	}
}

func TestScoreRejectsMalformed(t *testing.T) {
	for name, raw := range map[string]string{
		"not json":          `{"objectiveScores":`,
		"missing objective": `{"objectiveScores":{"a":90},"evidence":[],"feedback":"x"}`,
		"out of range":      `{"objectiveScores":{"a":90,"b":120},"evidence":[],"feedback":"x"}`,
		"negative":          `{"objectiveScores":{"a":90,"b":-1},"evidence":[],"feedback":"x"}`,
		"string score":      `{"objectiveScores":{"a":90,"b":"high"},"evidence":[],"feedback":"x"}`,
		"no feedback":       `{"objectiveScores":{"a":90,"b":90},"evidence":[],"feedback":"  "}`,
		"no scores":         `{"evidence":[],"feedback":"x"}`,
	} {
		if _, err := Score([]byte(raw), node); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: err = %v, want ErrMalformed", name, err)
		}
	}
}

func TestValidateAnswers(t *testing.T) {
	items := []curriculum.AssessmentItem{{ID: "1"}, {ID: "2"}}
	if err := ValidateAnswers(items, map[string]string{"1": "x", "2": "y"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAnswers(items, map[string]string{"1": "x"}); err == nil {
		t.Error("missing answer should fail")
	}
	if err := ValidateAnswers(items, map[string]string{"1": "x", "2": "y", "3": "z"}); err == nil {
		t.Error("unknown item should fail")
	}
	long := make([]byte, MaxAnswerRunes+1)
	for i := range long {
		long[i] = 'a'
	}
	if err := ValidateAnswers(items, map[string]string{"1": "x", "2": string(long)}); err == nil {
		t.Error("too-long answer should fail")
	}
}
