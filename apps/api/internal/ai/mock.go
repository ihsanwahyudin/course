package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/tutor"
)

// Mock is a deterministic, offline provider. It walks through the lesson's
// key points and practice prompts and grades answers by matching the
// curriculum's reference answers. Used for tests, e2e checks and local
// development without an API key.
type Mock struct {
	// Optional overrides for tests.
	TutorFunc func(TutorInput) (json.RawMessage, error)
	EvalFunc  func(EvalInput) (json.RawMessage, error)
}

func (m *Mock) Name() string { return "mock" }

type mockReply struct {
	Message          string   `json:"message"`
	Action           string   `json:"action"`
	ObservedMistakes []string `json:"observedMistakes"`
}

func (m *Mock) TutorReply(ctx context.Context, in TutorInput) (json.RawMessage, error) {
	if m.TutorFunc != nil {
		return m.TutorFunc(in)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	l := in.Lesson
	t := in.TurnsInPhase
	last := ""
	if n := len(in.Messages); n > 0 {
		last = in.Messages[n-1].Content
	}
	r := mockReply{Action: string(tutor.ActionContinue), ObservedMistakes: []string{}}

	switch in.Phase {
	case tutor.PhaseIntroduction:
		r.Message = "Great to have you here! In this lesson we will practise: " + objectiveList(in.Node) + ". " + enterPhase(l, tutor.PhaseExplanation)
		r.Action = string(tutor.ActionAdvancePhase)

	case tutor.PhaseExplanation:
		if t < len(l.KeyPoints) {
			r.Message = "Good. Next idea — " + keyPoint(l.KeyPoints[t])
		} else {
			r.Message = "You have the main ideas. " + enterPhase(l, tutor.PhaseGuidedPractice)
			r.Action = string(tutor.ActionAdvancePhase)
		}

	case tutor.PhaseGuidedPractice, tutor.PhaseIndependentPractice, tutor.PhaseReview:
		prompts := l.GuidedPractice
		next := tutor.PhaseIndependentPractice
		if in.Phase == tutor.PhaseIndependentPractice {
			prompts, next = l.IndependentPrompt, tutor.PhaseAssessment
		} else if in.Phase == tutor.PhaseReview {
			next = tutor.PhaseAssessment
		}
		answered := prompts[(t-1)%len(prompts)]
		var fb string
		if Matches(last, answered.Expected) {
			fb = "Correct, well done!"
		} else {
			fb = fmt.Sprintf("Not quite. Hint: %s A good answer is: %q.", answered.Hint, answered.Expected[0])
			r.Action = string(tutor.ActionAskRetry)
			r.ObservedMistakes = []string{answered.ObjectiveID}
		}
		if t >= len(prompts) && t >= 2 {
			r.Message = fb + " " + enterPhase(l, next)
			r.Action = string(tutor.ActionAdvancePhase)
		} else {
			r.Message = fb + " Next: " + prompts[t%len(prompts)].Prompt
		}

	default:
		r.Message = "Let's keep going."
	}
	return json.Marshal(r)
}

// enterPhase is the bridge text the tutor says when moving into a phase.
func enterPhase(l *curriculum.Lesson, p tutor.Phase) string {
	switch p {
	case tutor.PhaseExplanation:
		if len(l.KeyPoints) > 0 {
			return "First idea — " + keyPoint(l.KeyPoints[0])
		}
	case tutor.PhaseGuidedPractice:
		return "Let's practise together. " + l.GuidedPractice[0].Prompt
	case tutor.PhaseIndependentPractice:
		return "Now try on your own. " + l.IndependentPrompt[0].Prompt
	case tutor.PhaseAssessment:
		return "Great practice — you're ready for the assessment."
	}
	return ""
}

func keyPoint(k curriculum.KeyPoint) string {
	s := k.Title + ": " + k.Explanation
	if len(k.Examples) > 0 {
		s += fmt.Sprintf(" Example: %q (%s).", k.Examples[0].EN, k.Examples[0].ID)
	}
	return s + " Reply when you're ready to continue."
}

func objectiveList(n *curriculum.Node) string {
	parts := make([]string, len(n.Objectives))
	for i, o := range n.Objectives {
		parts[i] = strings.ToLower(o.Text[:1]) + o.Text[1:]
	}
	return strings.Join(parts, "; ")
}

func (m *Mock) Evaluate(ctx context.Context, in EvalInput) (json.RawMessage, error) {
	if m.EvalFunc != nil {
		return m.EvalFunc(in)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	sum := map[string]int{}
	count := map[string]int{}
	var evidence, misses []string
	for _, it := range in.Items {
		ans := in.Answers[it.ID]
		score := 0
		switch {
		case Matches(ans, it.AcceptableAnswers):
			score = 100
			evidence = append(evidence, fmt.Sprintf("Correct: %q", strings.TrimSpace(ans)))
		case strings.TrimSpace(ans) != "":
			score = 40
			misses = append(misses, it.ObjectiveID)
		default:
			misses = append(misses, it.ObjectiveID)
		}
		sum[it.ObjectiveID] += score
		count[it.ObjectiveID]++
	}
	scores := map[string]int{}
	for _, o := range in.Node.Objectives {
		if count[o.ID] > 0 {
			scores[o.ID] = sum[o.ID] / count[o.ID]
		}
	}
	feedback := "Excellent work — every answer met the objective."
	if len(misses) > 0 {
		feedback = "Review these objectives and try again: " + strings.Join(unique(misses), ", ") + "."
	}
	if evidence == nil {
		evidence = []string{"No fully correct answers yet."}
	}
	return json.Marshal(map[string]any{"objectiveScores": scores, "evidence": evidence, "feedback": feedback})
}

var nonWord = regexp.MustCompile(`[^a-z0-9' ]+`)

// Normalize lowercases, strips punctuation and collapses whitespace.
func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("’", "'", "‘", "'").Replace(s)
	s = nonWord.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

// Matches reports whether answer equals or contains one of the expected
// answers (as whole words), after normalization.
func Matches(answer string, expected []string) bool {
	a := " " + Normalize(answer) + " "
	if strings.TrimSpace(a) == "" {
		return false
	}
	for _, e := range expected {
		ne := Normalize(e)
		if ne != "" && strings.Contains(a, " "+ne+" ") {
			return true
		}
	}
	return false
}

func unique(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
