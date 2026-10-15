// Package tutor holds the lesson states and the contract for AI-built lesson
// plans: a lesson is a short sequence of steps (concept, examples, questions,
// …) shown one at a time. The AI writes the plan; this package validates it,
// and only the server checks answers and decides mastery.
package tutor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type Phase string

const (
	// PhaseLearning: working through the steps and the checkpoint.
	PhaseLearning Phase = "LEARNING"
	// PhaseMastered: the checkpoint was passed; the session is closed.
	PhaseMastered Phase = "MASTERED"
)

type StepKind string

const (
	StepConcept    StepKind = "concept"
	StepExamples   StepKind = "examples"
	StepQuestion   StepKind = "question"
	StepTimeline   StepKind = "timeline"
	StepFlashcards StepKind = "flashcards"
	StepArtifact   StepKind = "artifact"
	StepResources  StepKind = "resources" // filled by the server from curated links
)

var stepKinds = map[StepKind]bool{StepConcept: true, StepExamples: true, StepQuestion: true, StepTimeline: true, StepFlashcards: true, StepArtifact: true, StepResources: true}

const (
	QuestionMCQ       = "mcq"
	QuestionWordOrder = "word_order"
)

type Pair struct {
	EN string `json:"en"`
	ID string `json:"id"`
}

type Example struct {
	EN string `json:"en"`
	ID string `json:"id"`
}

// Question is a practice question. Answer, Solution and Explanation stay on
// the server until the learner has answered (or given up).
type Question struct {
	Type        string   `json:"type"` // mcq | word_order
	ObjectiveID string   `json:"objectiveId"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options"`  // mcq
	Answer      int      `json:"answer"`   // mcq: index into Options
	Tokens      []string `json:"tokens"`   // word_order: words shown, shuffled
	Solution    string   `json:"solution"` // word_order: the correct sentence
	Explanation string   `json:"explanation"`
	// Listen is English the learner hears (and doesn't see) before answering:
	// listening and pronunciation questions. Shown after answering.
	Listen string `json:"listen,omitempty"`
}

type Mark struct {
	Label string `json:"label"`
	Kind  string `json:"kind"` // past | now | future
}

type Timeline struct {
	Title   string `json:"title"`
	Caption string `json:"caption"`
	Marks   []Mark `json:"marks"`
}

type Card struct {
	Front string `json:"front"`
	Back  string `json:"back"`
}

// Artifact is a longer, saveable reference such as a cheat sheet.
type Artifact struct {
	Title    string     `json:"title"`
	Intro    string     `json:"intro"`
	Headers  []string   `json:"headers"`
	Rows     [][]string `json:"rows"`
	Tips     []string   `json:"tips"`
	Examples []Example  `json:"examples"`
}

type Step struct {
	Kind       StepKind  `json:"kind"`
	Title      string    `json:"title"`
	Text       string    `json:"text"` // the tutor's message for this step
	Body       string    `json:"body"` // concept explanation
	Pairs      []Pair    `json:"pairs"`
	Examples   []Example `json:"examples"`
	Question   *Question `json:"question"`
	Timeline   *Timeline `json:"timeline"`
	Flashcards []Card    `json:"flashcards"`
	Artifact   *Artifact `json:"artifact"`
}

type Plan struct {
	Greeting string `json:"greeting"`
	Steps    []Step `json:"steps"`
}

// ErrMalformed marks provider output that failed validation.
var ErrMalformed = errors.New("malformed tutor output")

const (
	MinSteps      = 4
	MaxSteps      = 12
	maxText       = 1200
	maxShort      = 200
	maxPairs      = 26 // a whole alphabet fits on one card
	maxExamples   = 8
	maxCards      = 8
	maxRows       = 26
	maxTokens     = 12
	MaxHints      = 2
	maxHintRunes  = 400
	maxAnswerText = 1500
)

func runes(s string) int { return utf8.RuneCountInString(s) }

func clean(s string) string { return strings.TrimSpace(s) }

func badLen(s string, max int) bool { return s == "" || runes(s) > max }

// NormalizeWords lowercases and strips punctuation for comparing sentences.
func NormalizeWords(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("’", "'", "‘", "'").Replace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\'' || r == ' ' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127:
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// ValidateQuestion checks one question against the lesson's objectives and
// returns a cleaned copy.
func ValidateQuestion(q Question, objectives map[string]bool) (Question, error) {
	return validateQuestion(q, objectives, maxShort*2)
}

func validateQuestion(q Question, objectives map[string]bool, maxPrompt int) (Question, error) {
	q.Prompt, q.Explanation, q.ObjectiveID = clean(q.Prompt), clean(q.Explanation), clean(q.ObjectiveID)
	if badLen(q.Prompt, maxPrompt) || badLen(q.Explanation, maxText) {
		return q, fmt.Errorf("question needs a prompt and an explanation")
	}
	if !objectives[q.ObjectiveID] {
		return q, fmt.Errorf("question targets unknown objective %q", q.ObjectiveID)
	}
	q.Listen = clean(q.Listen)
	if runes(q.Listen) > 300 {
		return q, fmt.Errorf("listen text must be at most 300 characters")
	}
	if q.Listen != "" && strings.Contains(NormalizeWords(q.Prompt), NormalizeWords(q.Listen)) {
		return q, fmt.Errorf("the prompt must not show what the learner should hear")
	}
	switch q.Type {
	case QuestionMCQ:
		if len(q.Options) < 2 || len(q.Options) > 4 || q.Answer < 0 || q.Answer >= len(q.Options) {
			return q, fmt.Errorf("mcq needs 2–4 options and an answer in range")
		}
		seen := map[string]bool{}
		for i, o := range q.Options {
			o = clean(o)
			if badLen(o, maxShort) || seen[strings.ToLower(o)] {
				return q, fmt.Errorf("mcq options must be distinct and non-empty")
			}
			seen[strings.ToLower(o)] = true
			q.Options[i] = o
		}
		q.Tokens, q.Solution = nil, ""
	case QuestionWordOrder:
		if len(q.Tokens) < 2 || len(q.Tokens) > maxTokens {
			return q, fmt.Errorf("word order needs 2–%d tokens", maxTokens)
		}
		for i, t := range q.Tokens {
			q.Tokens[i] = clean(t)
			if badLen(q.Tokens[i], 40) {
				return q, fmt.Errorf("word order tokens must be short words")
			}
		}
		q.Solution = clean(q.Solution)
		if !sameWords(q.Tokens, q.Solution) {
			return q, fmt.Errorf("word order tokens must spell the solution")
		}
		q.Options, q.Answer = nil, 0
	default:
		return q, fmt.Errorf("unknown question type %q", q.Type)
	}
	return q, nil
}

func sameWords(tokens []string, solution string) bool {
	a := strings.Fields(NormalizeWords(strings.Join(tokens, " ")))
	b := strings.Fields(NormalizeWords(solution))
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ParsePlan decodes and validates a plan. Every objective must be practised
// by at least one question.
func ParsePlan(raw []byte, objectives []string) (Plan, error) {
	var p Plan
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&p); err != nil {
		return Plan{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if err := ValidatePlan(&p, objectives); err != nil {
		return Plan{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	return p, nil
}

// ValidatePlan checks and cleans a plan in place.
func ValidatePlan(p *Plan, objectives []string) error {
	objs := map[string]bool{}
	for _, o := range objectives {
		objs[o] = true
	}
	p.Greeting = clean(p.Greeting)
	if runes(p.Greeting) > maxText {
		return fmt.Errorf("greeting too long")
	}
	if len(p.Steps) < MinSteps || len(p.Steps) > MaxSteps {
		return fmt.Errorf("plan needs %d–%d steps, got %d", MinSteps, MaxSteps, len(p.Steps))
	}
	covered := map[string]bool{}
	for i := range p.Steps {
		s := &p.Steps[i]
		s.Title, s.Text, s.Body = clean(s.Title), clean(s.Text), clean(s.Body)
		if !stepKinds[s.Kind] {
			return fmt.Errorf("step %d: unknown kind %q", i, s.Kind)
		}
		if badLen(s.Title, maxShort) || runes(s.Text) > maxText || runes(s.Body) > maxText {
			return fmt.Errorf("step %d: needs a short title and bounded text", i)
		}
		switch s.Kind {
		case StepConcept:
			if s.Body == "" || len(s.Pairs) > maxPairs {
				return fmt.Errorf("step %d: concept needs a body and at most %d pairs", i, maxPairs)
			}
		case StepExamples:
			if len(s.Examples) == 0 || len(s.Examples) > maxExamples {
				return fmt.Errorf("step %d: examples needs 1–%d sentences", i, maxExamples)
			}
		case StepQuestion:
			if s.Question == nil {
				return fmt.Errorf("step %d: question missing", i)
			}
			q, err := ValidateQuestion(*s.Question, objs)
			if err != nil {
				return fmt.Errorf("step %d: %v", i, err)
			}
			s.Question = &q
			covered[q.ObjectiveID] = true
		case StepTimeline:
			if s.Timeline == nil || len(s.Timeline.Marks) < 2 || len(s.Timeline.Marks) > 7 {
				return fmt.Errorf("step %d: timeline needs 2–7 marks", i)
			}
		case StepFlashcards:
			if len(s.Flashcards) == 0 || len(s.Flashcards) > maxCards {
				return fmt.Errorf("step %d: flashcards needs 1–%d cards", i, maxCards)
			}
			for _, c := range s.Flashcards {
				if badLen(clean(c.Front), 160) || badLen(clean(c.Back), 160) {
					return fmt.Errorf("step %d: flashcards need a front and back", i)
				}
			}
		case StepArtifact:
			a := s.Artifact
			if a == nil || badLen(clean(a.Title), maxShort) || len(a.Rows) > maxRows || len(a.Examples) > maxExamples {
				return fmt.Errorf("step %d: artifact needs a title and bounded content", i)
			}
			for _, r := range a.Rows {
				if len(r) != len(a.Headers) {
					return fmt.Errorf("step %d: artifact rows must match headers", i)
				}
			}
		}
		// Keep only the payload that belongs to the step's kind.
		if s.Kind != StepQuestion {
			s.Question = nil
		}
		if s.Kind != StepTimeline {
			s.Timeline = nil
		}
		if s.Kind != StepArtifact {
			s.Artifact = nil
		}
	}
	for _, o := range objectives {
		if !covered[o] {
			return fmt.Errorf("objective %q has no question", o)
		}
	}
	return nil
}

// Hint is the AI's nudge for a question.
type Hint struct {
	Hint string `json:"hint"`
}

// LeaksAnswer reports whether a hint gives the answer away.
func LeaksAnswer(hint string, q Question) bool {
	h := " " + NormalizeWords(hint) + " "
	var answer string
	switch q.Type {
	case QuestionMCQ:
		answer = NormalizeWords(q.Options[q.Answer])
		// A short answer like "he" counts only as a whole word.
		return answer != "" && strings.Contains(h, " "+answer+" ")
	case QuestionWordOrder:
		answer = NormalizeWords(q.Solution)
		return answer != "" && strings.Contains(h, answer)
	}
	return false
}

func ParseHint(raw []byte, q Question) (string, error) {
	var h Hint
	if err := json.Unmarshal(raw, &h); err != nil {
		return "", fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	h.Hint = clean(h.Hint)
	if badLen(h.Hint, maxHintRunes) {
		return "", fmt.Errorf("%w: empty or long hint", ErrMalformed)
	}
	if LeaksAnswer(h.Hint, q) {
		return "", fmt.Errorf("%w: hint gives the answer away", ErrMalformed)
	}
	return h.Hint, nil
}

// Scope of a learner's question relative to the lesson.
const (
	ScopeIn          = "in"
	ScopeOtherLesson = "other_lesson"
	ScopeOffTopic    = "off_topic"
)

// Answer is the AI's reply to "Tanya AI".
type Answer struct {
	Answer      string `json:"answer"`
	Scope       string `json:"scope"`
	OtherLesson string `json:"otherLesson"` // title of the lesson that covers it
	Remember    string `json:"remember"`    // a fact worth remembering about the learner, or ""
}

func ParseAnswer(raw []byte) (Answer, error) {
	var a Answer
	if err := json.Unmarshal(raw, &a); err != nil {
		return Answer{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	a.Answer, a.OtherLesson, a.Remember = clean(a.Answer), clean(a.OtherLesson), clean(a.Remember)
	if badLen(a.Answer, maxAnswerText) {
		return Answer{}, fmt.Errorf("%w: empty or long answer", ErrMalformed)
	}
	switch a.Scope {
	case ScopeIn, ScopeOffTopic:
		a.OtherLesson = ""
	case ScopeOtherLesson:
		if runes(a.OtherLesson) > maxShort {
			a.OtherLesson = ""
		}
	default:
		a.Scope = ScopeIn
	}
	if runes(a.Remember) > maxShort {
		a.Remember = ""
	}
	return a, nil
}

// Summary is written after the checkpoint and carried to the next lesson.
type Summary struct {
	Mastered    []string `json:"mastered"`    // short phrases
	Review      []string `json:"review"`      // what still needs work
	Memory      []string `json:"memory"`      // learning patterns worth remembering
	NextOpening string   `json:"nextOpening"` // first line for the next lesson
}

func ParseSummary(raw []byte) (Summary, error) {
	var s Summary
	if err := json.Unmarshal(raw, &s); err != nil {
		return Summary{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	trim := func(xs []string, n int) []string {
		out := []string{}
		for _, x := range xs {
			if x = clean(x); x != "" && runes(x) <= maxShort && len(out) < n {
				out = append(out, x)
			}
		}
		return out
	}
	s.Mastered, s.Review, s.Memory = trim(s.Mastered, 5), trim(s.Review, 5), trim(s.Memory, 3)
	s.NextOpening = clean(s.NextOpening)
	if runes(s.NextOpening) > maxText {
		s.NextOpening = ""
	}
	return s, nil
}
