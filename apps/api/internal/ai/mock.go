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

// Mock is a deterministic, offline provider. It answers from the lesson's
// key points, builds quizzes and flashcards from the lesson's practice data,
// and grades answers by matching the curriculum's reference answers. Used
// for tests, e2e checks and local development without an API key.
type Mock struct {
	// Optional overrides for tests.
	TutorFunc func(TutorInput) (json.RawMessage, error)
	EvalFunc  func(EvalInput) (json.RawMessage, error)
	// Calls counts provider calls (tests check caching and pre-grading).
	Calls struct{ Tutor, Eval, Search int }
}

func (m *Mock) Name() string { return "mock" }

// mockUsage is a fixed, plausible token count so usage accounting can be tested.
var mockUsage = Usage{Input: 1000, Cached: 0, Output: 100}

func (m *Mock) TutorReply(ctx context.Context, in TutorInput) (Result, error) {
	m.Calls.Tutor++
	if m.TutorFunc != nil {
		raw, err := m.TutorFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	raw, err := json.Marshal(mockReply(in))
	return Result{Raw: raw, Usage: mockUsage}, err
}

func (m *Mock) WebSearch(ctx context.Context, in SearchInput) ([]Link, Usage, error) {
	m.Calls.Search++
	return nil, Usage{}, ErrNoSearch
}

func mockReply(in TutorInput) tutor.Reply {
	l := in.Lesson
	r := tutor.Reply{Quiz: []tutor.QuizQuestion{}, Flashcards: []tutor.Card{}}
	last := ""
	if n := len(in.Messages); n > 0 {
		last = in.Messages[n-1].Content
	}
	switch in.Intent {
	case tutor.IntentQuiz:
		r.Message = say(in, "quiz")
		r.Quiz = mockQuiz(l)
	case tutor.IntentFlashcards:
		r.Message = say(in, "cards")
		for _, k := range l.KeyPoints {
			for _, e := range k.Examples {
				if len(r.Flashcards) < tutor.MaxFlashcards {
					r.Flashcards = append(r.Flashcards, tutor.Card{Front: e.ID, Back: e.EN})
				}
			}
		}
	case tutor.IntentExamples, tutor.IntentVideo, tutor.IntentImage:
		r.Message = say(in, string(in.Intent))
		r.Search = tutor.Search{Kind: in.Intent.SearchKind(), Query: in.Node.Title}
	case tutor.IntentSimpler:
		r.Message = say(in, "simpler") + keyPoint(in, l.KeyPoints[0])
	default:
		r.Message = say(in, "about") + keyPoint(in, bestKeyPoint(l, last))
	}
	return r
}

// bestKeyPoint picks the key point sharing the most words with the question.
func bestKeyPoint(l *curriculum.Lesson, question string) curriculum.KeyPoint {
	words := strings.Fields(Normalize(question))
	best, bestScore := l.KeyPoints[0], 0
	for _, k := range l.KeyPoints {
		text := " " + Normalize(k.Title+" "+k.Explanation) + " "
		score := 0
		for _, w := range words {
			if len(w) > 2 && strings.Contains(text, " "+w+" ") {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = k, score
		}
	}
	return best
}

// mockQuiz turns practice prompts into multiple-choice questions, using the
// other prompts' answers as distractors.
func mockQuiz(l *curriculum.Lesson) []tutor.QuizQuestion {
	prompts := append(append([]curriculum.PracticePrompt{}, l.GuidedPractice...), l.IndependentPrompt...)
	var answers []string
	for _, p := range prompts {
		if len(p.Expected) > 0 {
			answers = append(answers, p.Expected[0])
		}
	}
	var out []tutor.QuizQuestion
	for i, p := range prompts {
		if len(p.Expected) == 0 || len(out) >= 4 {
			continue
		}
		opts := []string{p.Expected[0]}
		for j := 1; j < len(answers) && len(opts) < 3; j++ {
			d := answers[(i+j)%len(answers)]
			if Normalize(d) != Normalize(p.Expected[0]) && !contains(opts, d) {
				opts = append(opts, d)
			}
		}
		if len(opts) < 2 {
			continue
		}
		// Rotate so the right answer isn't always first.
		k := i % len(opts)
		opts = append(opts[k:], opts[:k]...)
		out = append(out, tutor.QuizQuestion{Prompt: p.Prompt, Options: opts, Answer: (len(opts) - k) % len(opts), Explain: p.Hint})
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func keyPoint(in TutorInput, k curriculum.KeyPoint) string {
	s := k.Title + ": " + k.Explanation
	if len(k.Examples) > 0 {
		s += fmt.Sprintf(say(in, "example"), k.Examples[0].EN, k.Examples[0].ID)
	}
	return s
}

// Mock phrasing per UI language. Lesson content stays as written.
var mockText = map[string]map[string]string{
	"en": {
		"about": "Here's the key idea. ", "simpler": "Let's make it simple. ", "example": " Example: %q (%s).",
		"quiz": "Here's a short practice quiz. It doesn't count towards mastery.", "cards": "Here are some flashcards for this lesson.",
		"examples": "Here's where people use this in real life.", "video": "Here are videos that show this in action.",
		"image":       "Here are some pictures to help.",
		"all_correct": "Excellent work — every answer met the objective.", "review": "Review and try again: %s.",
		"correct_answer": "Correct: %q", "none_correct": "No fully correct answers yet.",
	},
	"id": {
		"about": "Ini ide utamanya. ", "simpler": "Kita buat sederhana. ", "example": " Contoh: %q (%s).",
		"quiz": "Ini kuis latihan singkat. Tidak memengaruhi status dikuasai.", "cards": "Ini beberapa flashcard untuk pelajaran ini.",
		"examples": "Ini contoh pemakaiannya di kehidupan nyata.", "video": "Ini video yang menunjukkan penerapannya.",
		"image":       "Ini beberapa gambar yang bisa membantu.",
		"all_correct": "Kerja bagus — semua jawaban memenuhi tujuan pelajaran.", "review": "Ulas lagi lalu coba kembali: %s.",
		"correct_answer": "Benar: %q", "none_correct": "Belum ada jawaban yang sepenuhnya benar.",
	},
}

func say(in TutorInput, key string) string { return phrase(in.Locale, key) }

func phrase(locale, key string) string {
	if m, ok := mockText[locale]; ok {
		return m[key]
	}
	return mockText["en"][key]
}

func (m *Mock) Evaluate(ctx context.Context, in EvalInput) (Result, error) {
	m.Calls.Eval++
	if m.EvalFunc != nil {
		raw, err := m.EvalFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
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
			evidence = append(evidence, fmt.Sprintf(phrase(in.Locale, "correct_answer"), strings.TrimSpace(ans)))
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
	feedback := phrase(in.Locale, "all_correct")
	if len(misses) > 0 {
		var texts []string
		for _, id := range unique(misses) {
			for _, o := range in.Node.Objectives {
				if o.ID == id {
					texts = append(texts, strings.ToLower(o.Text[:1])+o.Text[1:])
				}
			}
		}
		feedback = fmt.Sprintf(phrase(in.Locale, "review"), strings.Join(texts, "; "))
	}
	if evidence == nil {
		evidence = []string{phrase(in.Locale, "none_correct")}
	}
	raw, err := json.Marshal(map[string]any{"objectiveScores": scores, "evidence": evidence, "feedback": feedback})
	return Result{Raw: raw, Usage: mockUsage}, err
}

var nonWord = regexp.MustCompile(`[^a-z0-9' ]+`)

// Normalize lowercases, strips punctuation and collapses whitespace.
func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("’", "'", "‘", "'").Replace(s)
	s = nonWord.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

// ExactMatch reports whether answer equals one of the expected answers
// after normalization (used to score answers without calling the AI).
func ExactMatch(answer string, expected []string) bool {
	a := Normalize(answer)
	if a == "" {
		return false
	}
	for _, e := range expected {
		if Normalize(e) == a {
			return true
		}
	}
	return false
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
