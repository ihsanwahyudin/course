// Package assessment validates evaluator output and applies the scoring rules.
// The evaluator only proposes per-objective scores; the pass/fail decision is
// made here, deterministically.
package assessment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"fluentpath/api/internal/curriculum"
)

var ErrMalformed = errors.New("malformed evaluator result")

const MaxAnswerRunes = 1000

type Result struct {
	ObjectiveScores map[string]int `json:"objectiveScores"`
	Evidence        []string       `json:"evidence"`
	Feedback        string         `json:"feedback"`
	Score           int            `json:"score"`
	Threshold       int            `json:"threshold"`
	Passed          bool           `json:"passed"`
	WeakObjectives  []string       `json:"weakObjectives"`
}

type rawResult struct {
	ObjectiveScores map[string]json.Number `json:"objectiveScores"`
	Evidence        []string               `json:"evidence"`
	Feedback        string                 `json:"feedback"`
}

// Score validates raw evaluator JSON against the node's objectives and
// computes the overall score as the mean of objective scores.
func Score(raw []byte, node *curriculum.Node) (Result, error) {
	var rr rawResult
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&rr); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if rr.ObjectiveScores == nil {
		return Result{}, fmt.Errorf("%w: missing objectiveScores", ErrMalformed)
	}
	res := Result{ObjectiveScores: map[string]int{}, Threshold: node.MasteryThreshold}
	total := 0
	for _, o := range node.Objectives {
		n, ok := rr.ObjectiveScores[o.ID]
		if !ok {
			return Result{}, fmt.Errorf("%w: no score for objective %q", ErrMalformed, o.ID)
		}
		f, err := n.Float64()
		if err != nil || math.IsNaN(f) || f < 0 || f > 100 {
			return Result{}, fmt.Errorf("%w: objective %q score %q out of range", ErrMalformed, o.ID, n)
		}
		v := int(math.Round(f))
		res.ObjectiveScores[o.ID] = v
		total += v
		if v < node.MasteryThreshold {
			res.WeakObjectives = append(res.WeakObjectives, o.ID)
		}
	}
	res.Score = int(math.Round(float64(total) / float64(len(node.Objectives))))
	res.Passed = res.Score >= node.MasteryThreshold

	res.Feedback = strings.TrimSpace(rr.Feedback)
	if res.Feedback == "" {
		return Result{}, fmt.Errorf("%w: empty feedback", ErrMalformed)
	}
	if utf8.RuneCountInString(res.Feedback) > 1500 {
		res.Feedback = string([]rune(res.Feedback)[:1500])
	}
	for _, e := range rr.Evidence {
		if e = strings.TrimSpace(e); e != "" && len(res.Evidence) < 8 {
			res.Evidence = append(res.Evidence, e)
		}
	}
	if res.Evidence == nil {
		res.Evidence = []string{}
	}
	if res.WeakObjectives == nil {
		res.WeakObjectives = []string{}
	}
	return res, nil
}

// ValidateAnswers checks that every item has a non-empty, bounded answer and
// that no unknown items were submitted.
func ValidateAnswers(items []curriculum.AssessmentItem, answers map[string]string) error {
	known := map[string]bool{}
	for _, it := range items {
		known[it.ID] = true
		a := strings.TrimSpace(answers[it.ID])
		if a == "" {
			return fmt.Errorf("answer required for item %q", it.ID)
		}
		if utf8.RuneCountInString(a) > MaxAnswerRunes {
			return fmt.Errorf("answer for item %q is too long", it.ID)
		}
	}
	for id := range answers {
		if !known[id] {
			return fmt.Errorf("unknown item %q", id)
		}
	}
	return nil
}
