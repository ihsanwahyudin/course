// Package ai adapts AI providers to the jobs FluentPath gives them: the
// lesson helper, rubric-based assessment and real-world link search.
// Providers return raw JSON; the tutor and assessment packages validate it
// before anything is stored, and nothing a provider returns can change
// mastery directly.
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

// ErrNoSearch means the provider can't search the web.
var ErrNoSearch = errors.New("web search not available")

type ChatMessage struct {
	Role    string `json:"role"` // user | assistant
	Content string `json:"content"`
}

// Usage is the token count of one provider call. Cached input tokens are
// part of Input and billed at a discount.
type Usage struct {
	Input  int `json:"input"`
	Cached int `json:"cached"`
	Output int `json:"output"`
}

func (u Usage) Add(o Usage) Usage {
	return Usage{u.Input + o.Input, u.Cached + o.Cached, u.Output + o.Output}
}

// Result is a provider's raw JSON answer plus what it cost.
type Result struct {
	Raw   json.RawMessage
	Usage Usage
}

type TutorInput struct {
	Policy         string // course.json tutor.policy
	Locale         string // language the lesson runs in (en | id)
	LanguagePolicy string // course.json tutor.languagePolicy[Locale]
	CurrentLevel   string
	TargetLevel    string
	Node           *curriculum.Node   // objectives in the lesson's language
	Lesson         *curriculum.Lesson // material and key points (English)
	// LearnerProfile holds author-written hints chosen by the learner's
	// onboarding answers; LearnerNotes is what they typed themselves (data).
	LearnerProfile []string
	LearnerNotes   []string
	// Dynamic, per-turn context (kept after the cacheable prefix).
	Intent         tutor.Intent
	WeakObjectives []string // from the last failed final quiz
	Messages       []ChatMessage
	// CacheKey groups requests that share a prompt prefix (course, node,
	// language) so the provider can reuse its prompt cache.
	CacheKey string
}

type EvalInput struct {
	Policy         string // course.json tutor.evaluatorPolicy
	Locale         string // language for feedback and evidence
	LanguagePolicy string
	TargetLevel    string
	Node           *curriculum.Node
	Items          []curriculum.AssessmentItem
	Answers        map[string]string // item id -> learner answer
	// GoalHint is the learner's goal, for relating feedback to it. It must
	// not affect scores.
	GoalHint string
}

// SearchInput asks for real-world links about a lesson.
type SearchInput struct {
	Kind        string // tutor.KindVideo | tutor.KindArticle
	Query       string
	LessonTitle string
	Locale      string
}

// Link is a search result. Providers only return URLs their search tool
// actually cited.
type Link struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Source string `json:"source"`
}

type Provider interface {
	Name() string
	TutorReply(ctx context.Context, in TutorInput) (Result, error)
	Evaluate(ctx context.Context, in EvalInput) (Result, error)
	// WebSearch returns ErrNoSearch when the provider has no search tool.
	WebSearch(ctx context.Context, in SearchInput) ([]Link, Usage, error)
}
