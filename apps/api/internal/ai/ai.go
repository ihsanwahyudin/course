// Package ai adapts AI providers to the jobs FluentPath gives them: writing a
// lesson plan, hints, similar questions, answering questions inside a step,
// summarising a finished lesson, rubric-based assessment, and speech
// (text-to-speech and transcription). Providers return raw JSON; the tutor
// and assessment packages validate it before anything is stored, and nothing
// a provider returns can change mastery directly.
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

// ErrNoAudio means the provider can't do speech; clients fall back to the
// device's own voice.
var ErrNoAudio = errors.New("speech not available")

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

// Result is a provider's raw JSON answer plus what it cost.
type Result struct {
	Raw   json.RawMessage
	Usage Usage
}

// LessonInput is the shared, cacheable context for every lesson call.
type LessonInput struct {
	Policy         string // course.json tutor.policy
	Locale         string // en | id
	LanguagePolicy string // course.json tutor.languagePolicy[Locale]
	CurrentLevel   string
	TargetLevel    string
	Node           *curriculum.Node         // objectives in the lesson's language
	Lesson         *curriculum.Lesson       // English material and practice
	Material       *curriculum.MaterialView // material in the lesson's language
	LearnerProfile []string                 // author-written hints from onboarding answers
	LearnerNotes   []string                 // the learner's own words (data)
	Memory         []string                 // what the tutor remembers about the learner
	CacheKey       string
}

type HintInput struct {
	LessonInput
	Question tutor.Question
	Previous []string // hints already given
}

type SimilarInput struct {
	LessonInput
	Question tutor.Question // the one the learner got wrong
}

type AskInput struct {
	LessonInput
	Step         tutor.Step
	Question     string
	History      []ChatMessage // earlier questions in this step
	OtherLessons []string      // titles of the course's other lessons, for redirects
}

type SummaryInput struct {
	LessonInput
	Passed     bool
	Scores     map[string]int // objective id -> score
	Mistakes   []string       // practice questions the learner got wrong
	NextLesson string         // title of the next lesson, if any
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

type Provider interface {
	Name() string
	LessonPlan(ctx context.Context, in LessonInput) (Result, error)
	Hint(ctx context.Context, in HintInput) (Result, error)
	SimilarQuestion(ctx context.Context, in SimilarInput) (Result, error)
	Ask(ctx context.Context, in AskInput) (Result, error)
	Summarize(ctx context.Context, in SummaryInput) (Result, error)
	Evaluate(ctx context.Context, in EvalInput) (Result, error)
	// Speech returns MP3 audio, or ErrNoAudio.
	Speech(ctx context.Context, text string) ([]byte, error)
	// Transcribe turns a short English recording into text, or ErrNoAudio.
	Transcribe(ctx context.Context, audio []byte, filename string) (string, error)
}
