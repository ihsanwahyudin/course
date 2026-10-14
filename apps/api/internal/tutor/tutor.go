// Package tutor holds the lesson states and the AI helper's reply contract.
// Lessons are reading material with an AI helper beside it; the helper can
// explain, quiz, make flashcards and look up real-world examples, but only
// the server decides mastery (through the final quiz).
package tutor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Phase string

const (
	// PhaseLearning: reading, asking the helper, taking the final quiz.
	PhaseLearning Phase = "LEARNING"
	// PhaseMastered: the final quiz was passed; the session is closed.
	PhaseMastered Phase = "MASTERED"
)

// Intent is what the learner asked the helper for. Quick-action buttons send
// one; typed messages are "ask".
type Intent string

const (
	IntentAsk        Intent = "ask"
	IntentSimpler    Intent = "simpler"
	IntentQuiz       Intent = "quiz"
	IntentFlashcards Intent = "flashcards"
	IntentExamples   Intent = "examples"
	IntentVideo      Intent = "video"
	IntentImage      Intent = "image"
)

var intents = map[Intent]bool{IntentAsk: true, IntentSimpler: true, IntentQuiz: true, IntentFlashcards: true,
	IntentExamples: true, IntentVideo: true, IntentImage: true}

func (i Intent) Valid() bool { return intents[i] }

// SearchKind returns the resource kind an intent looks up, if any.
func (i Intent) SearchKind() string {
	switch i {
	case IntentVideo:
		return KindVideo
	case IntentImage:
		return KindImage
	case IntentExamples:
		return KindArticle
	}
	return ""
}

// MaxOutputTokens keeps each kind of reply short.
func (i Intent) MaxOutputTokens() int {
	switch i {
	case IntentQuiz:
		return 700
	case IntentFlashcards:
		return 500
	case IntentExamples, IntentVideo, IntentImage:
		return 300
	}
	return 350
}

const (
	KindVideo   = "video"
	KindArticle = "article"
	KindImage   = "image"
)

// ErrMalformed marks provider output that failed validation.
var ErrMalformed = errors.New("malformed tutor reply")

const (
	maxReplyRunes  = 2000
	MaxQuiz        = 5
	MaxFlashcards  = 8
	maxPromptRunes = 300
	maxCardRunes   = 160
	maxQueryRunes  = 100
)

type QuizQuestion struct {
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
	Explain string   `json:"explain"`
}

type Card struct {
	Front string `json:"front"`
	Back  string `json:"back"`
}

type Search struct {
	Kind  string `json:"kind"` // "" | video | article | image
	Query string `json:"query"`
}

// Reply is the helper's structured answer.
type Reply struct {
	Message    string         `json:"message"`
	Quiz       []QuizQuestion `json:"quiz"`
	Flashcards []Card         `json:"flashcards"`
	Search     Search         `json:"search"`
}

func runes(s string) int { return utf8.RuneCountInString(s) }

// ParseReply validates a provider's raw JSON reply. A bad message rejects
// the reply; malformed quiz questions, cards or searches are dropped.
func ParseReply(raw []byte) (Reply, error) {
	var r Reply
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&r); err != nil {
		return Reply{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	r.Message = strings.TrimSpace(r.Message)
	if r.Message == "" {
		return Reply{}, fmt.Errorf("%w: empty message", ErrMalformed)
	}
	if runes(r.Message) > maxReplyRunes {
		return Reply{}, fmt.Errorf("%w: message too long", ErrMalformed)
	}
	quiz := []QuizQuestion{}
	for _, q := range r.Quiz {
		q.Prompt, q.Explain = strings.TrimSpace(q.Prompt), strings.TrimSpace(q.Explain)
		ok := q.Prompt != "" && runes(q.Prompt) <= maxPromptRunes && runes(q.Explain) <= maxPromptRunes &&
			len(q.Options) >= 2 && len(q.Options) <= 4 && q.Answer >= 0 && q.Answer < len(q.Options)
		for i, o := range q.Options {
			q.Options[i] = strings.TrimSpace(o)
			ok = ok && q.Options[i] != "" && runes(q.Options[i]) <= maxCardRunes
		}
		if ok && len(quiz) < MaxQuiz {
			quiz = append(quiz, q)
		}
	}
	r.Quiz = quiz
	cards := []Card{}
	for _, c := range r.Flashcards {
		c.Front, c.Back = strings.TrimSpace(c.Front), strings.TrimSpace(c.Back)
		if c.Front != "" && c.Back != "" && runes(c.Front) <= maxCardRunes && runes(c.Back) <= maxCardRunes && len(cards) < MaxFlashcards {
			cards = append(cards, c)
		}
	}
	r.Flashcards = cards
	r.Search.Query = strings.TrimSpace(r.Search.Query)
	switch r.Search.Kind {
	case KindVideo, KindArticle, KindImage:
		if r.Search.Query == "" || runes(r.Search.Query) > maxQueryRunes {
			r.Search = Search{}
		}
	default:
		r.Search = Search{}
	}
	return r, nil
}
