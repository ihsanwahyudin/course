// Package ai adapts AI providers to the two jobs FluentPath gives them:
// tutoring replies and rubric-based assessment. Providers return raw JSON; the
// tutor and assessment packages validate it before anything is stored, and
// nothing a provider returns can change mastery directly.
package ai

import (
	"context"
	"encoding/json"
	"errors"

	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/tutor"
)

// ErrUnavailable wraps provider failures (timeouts, 5xx, refusals,
// truncated output) that the client may retry.
var ErrUnavailable = errors.New("ai provider unavailable")

type ChatMessage struct {
	Role    string `json:"role"` // user | assistant
	Content string `json:"content"`
}

type TutorInput struct {
	Policy             string // course.json tutor.policy
	CurrentLevel       string
	TargetLevel        string
	Node               *curriculum.Node
	Lesson             *curriculum.Lesson
	Phase              tutor.Phase
	NextPhase          tutor.Phase
	TurnsInPhase       int // learner replies in this phase, including the newest
	Messages           []ChatMessage
	MasteredNodeTitles []string
	ObservedWeaknesses []string
}

type EvalInput struct {
	Policy      string // course.json tutor.evaluatorPolicy
	TargetLevel string
	Node        *curriculum.Node
	Items       []curriculum.AssessmentItem
	Answers     map[string]string // item id -> learner answer
}

type Provider interface {
	Name() string
	TutorReply(ctx context.Context, in TutorInput) (json.RawMessage, error)
	Evaluate(ctx context.Context, in EvalInput) (json.RawMessage, error)
}
