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

// Mock is a deterministic, offline provider: plans come straight from the
// lesson material, hints and answers are canned, checkpoints are the lesson's
// hand-written ones and grading matches their reference answers. Used for tests, e2e checks and local
// development without an API key.
type Mock struct {
	// Optional overrides for tests.
	PlanFunc       func(LessonInput) (json.RawMessage, error)
	HintFunc       func(HintInput) (json.RawMessage, error)
	AskFunc        func(AskInput) (json.RawMessage, error)
	CheckpointFunc func(CheckpointInput) (json.RawMessage, error)
	GradeFunc      func(GradeInput) (json.RawMessage, error)
	VerifyFunc     func(VerifyInput) (json.RawMessage, error)
	// Calls counts provider calls (tests check caching and pre-grading).
	Calls struct{ Plan, Hint, Similar, Ask, Summary, Checkpoint, Verify, Grade int }
}

func (m *Mock) Name() string { return "mock" }

// mockUsage is a fixed, plausible token count so usage accounting can be tested.
var mockUsage = Usage{Input: 1000, Cached: 0, Output: 100}

func ok(v any) (Result, error) {
	raw, err := json.Marshal(v)
	return Result{Raw: raw, Usage: mockUsage}, err
}

func (m *Mock) LessonPlan(ctx context.Context, in LessonInput) (Result, error) {
	m.Calls.Plan++
	if m.PlanFunc != nil {
		raw, err := m.PlanFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	p := tutor.FallbackPlan(in.Node, in.Lesson, in.Material, in.Locale)
	p.Greeting = phrase(in.Locale, "greeting")
	return ok(p)
}

func (m *Mock) Hint(ctx context.Context, in HintInput) (Result, error) {
	m.Calls.Hint++
	if m.HintFunc != nil {
		raw, err := m.HintFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	key := "hint1"
	if len(in.Previous) > 0 {
		key = "hint2"
	}
	return ok(tutor.Hint{Hint: phrase(in.Locale, key)})
}

func (m *Mock) SimilarQuestion(ctx context.Context, in SimilarInput) (Result, error) {
	m.Calls.Similar++
	l := in.Lesson
	prompts := append(append([]curriculum.PracticePrompt{}, l.GuidedPractice...), l.IndependentPrompt...)
	for i, p := range prompts {
		if p.ObjectiveID != in.Question.ObjectiveID || p.Prompt == in.Question.Prompt {
			continue
		}
		if q, okq := tutor.QuestionFromPrompt(p, prompts, i+1); okq {
			return ok(q)
		}
	}
	// Same question with the options in another order.
	q := in.Question
	if q.Type == tutor.QuestionMCQ && len(q.Options) > 1 {
		opts := append(append([]string{}, q.Options[1:]...), q.Options[0])
		q.Options, q.Answer = opts, (q.Answer-1+len(opts))%len(opts)
		if len(q.DisplayOptions) > 0 {
			q.DisplayOptions = append(append([]string{}, q.DisplayOptions[1:]...), q.DisplayOptions[0])
		}
	}
	return ok(q)
}

func (m *Mock) Ask(ctx context.Context, in AskInput) (Result, error) {
	m.Calls.Ask++
	if m.AskFunc != nil {
		raw, err := m.AskFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	k := bestKeyPoint(in.Lesson, in.Question)
	text := phrase(in.Locale, "about") + k.Title + ": " + k.Explanation
	if len(k.Examples) > 0 {
		text += fmt.Sprintf(phrase(in.Locale, "example"), k.Examples[0].EN, k.Examples[0].ID)
	}
	return ok(tutor.Answer{Answer: text, Scope: tutor.ScopeIn})
}

func (m *Mock) Summarize(ctx context.Context, in SummaryInput) (Result, error) {
	m.Calls.Summary++
	s := tutor.Summary{Mastered: []string{}, Review: []string{}, Memory: []string{}}
	for _, o := range in.Node.Objectives {
		if in.Scores[o.ID] >= 80 {
			s.Mastered = append(s.Mastered, o.Text)
		} else {
			s.Review = append(s.Review, o.Text)
			s.Memory = append(s.Memory, fmt.Sprintf(phrase(in.Locale, "needs"), lowerFirst(o.Text)))
		}
	}
	if in.NextLesson != "" {
		s.NextOpening = fmt.Sprintf(phrase(in.Locale, "next"), in.Node.Title, in.NextLesson)
	}
	return ok(s)
}

func (m *Mock) Speech(ctx context.Context, text string) ([]byte, error) { return nil, ErrNoAudio }

func (m *Mock) Transcribe(ctx context.Context, audio []byte, filename string) (string, error) {
	return "", ErrNoAudio
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

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// Mock phrasing per UI language. Lesson content stays as written.
var mockText = map[string]map[string]string{
	"en": {
		"greeting": "Hi! Let's go through this lesson one step at a time. Tap Next when you're ready, or ask me anything about it.",
		"hint1":    "Look at who or what the sentence is about first.", "hint2": "Compare each option with the rule on the concept card.",
		"about": "Here's the key idea. ", "example": " Example: %q (%s).", "needs": "Needs more practice with %s",
		"next":        "You've finished %s. Next we'll build on it in %s.",
		"all_correct": "Excellent work — every answer met the objective.", "review": "Review and try again: %s.",
		"correct_answer": "Correct: %q", "none_correct": "No fully correct answers yet.",
	},
	"id": {
		"greeting": "Hai! Kita jalani pelajaran ini satu langkah demi satu langkah. Tekan Lanjut kalau siap, atau tanyakan apa saja tentang pelajaran ini.",
		"hint1":    "Lihat dulu kalimat ini membicarakan siapa atau apa.", "hint2": "Bandingkan setiap pilihan dengan aturan di kartu konsep.",
		"about": "Ini ide utamanya. ", "example": " Contoh: %q (%s).", "needs": "Masih perlu latihan: %s",
		"next":        "Kamu sudah menyelesaikan %s. Berikutnya kita lanjutkan di %s.",
		"all_correct": "Kerja bagus — semua jawaban memenuhi tujuan pelajaran.", "review": "Ulas lagi lalu coba kembali: %s.",
		"correct_answer": "Benar: %q", "none_correct": "Belum ada jawaban yang sepenuhnya benar.",
	},
}

func phrase(locale, key string) string {
	if m, ok := mockText[locale]; ok {
		return m[key]
	}
	return mockText["en"][key]
}

func (m *Mock) Checkpoint(ctx context.Context, in CheckpointInput) (Result, error) {
	m.Calls.Checkpoint++
	if m.CheckpointFunc != nil {
		raw, err := m.CheckpointFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(in.LevelLessons) > 0 {
		return ok(map[string]any{"items": levelDrafts(in.LevelLessons)})
	}
	return ok(map[string]any{"items": tutor.AuthoredDrafts(in.Lesson, in.Locale)})
}

// VerifyCheckpoint passes every question unless a test says otherwise.
func (m *Mock) VerifyCheckpoint(ctx context.Context, in VerifyInput) (Result, error) {
	m.Calls.Verify++
	if m.VerifyFunc != nil {
		raw, err := m.VerifyFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	type check struct {
		ID      string `json:"id"`
		OK      bool   `json:"ok"`
		Problem string `json:"problem"`
	}
	out := make([]check, 0, len(in.Items))
	for _, it := range in.Items {
		out = append(out, check{it.ID, true, ""})
	}
	return ok(map[string]any{"items": out})
}

// AuthoredCheckpoint is a lesson's hand-written checkpoint (also the
// fallback when the AI's checkpoint doesn't validate). The course check
// makes sure it converts.
func AuthoredCheckpoint(l *curriculum.Lesson, lang string) []tutor.CheckItem {
	items, _ := tutor.AuthoredItems(l, lang)
	return items
}

func (m *Mock) GradeCheckpoint(ctx context.Context, in GradeInput) (Result, error) {
	m.Calls.Grade++
	if m.GradeFunc != nil {
		raw, err := m.GradeFunc(in)
		return Result{Raw: raw, Usage: mockUsage}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	type graded struct {
		ID            string `json:"id"`
		Score         int    `json:"score"`
		CorrectAnswer string `json:"correctAnswer"`
		Explanation   string `json:"explanation"`
	}
	var items []graded
	var strengths, misses []string
	for _, it := range in.Items {
		ans := strings.TrimSpace(in.Answers[it.ID])
		score, fixed := in.Fixed[it.ID]
		if !fixed {
			switch {
			case tutor.FormatSensitiveText(it):
				// The offline fallback must not erase the very punctuation
				// being assessed (for example commas in a house-style date).
				if tutor.MatchFormattedText(it, ans) {
					score = 100
				}
			case Matches(ans, it.Accepted):
				score = 100
			case ans != "":
				score = 40
			}
		}
		expl := fmt.Sprintf(phrase(in.Locale, "correct_answer"), it.Key())
		if score >= 80 {
			strengths = append(strengths, it.ObjectiveID)
		} else {
			misses = append(misses, it.ObjectiveID)
		}
		items = append(items, graded{it.ID, score, it.Key(), expl})
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
	return ok(map[string]any{"items": items, "strengths": unique(strengths), "toReview": unique(misses), "feedback": feedback})
}

var nonWord = regexp.MustCompile(`[^a-z0-9' ]+`)

// Normalize lowercases, strips punctuation and collapses whitespace.
func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(tutor.FoldAccents(s)))
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

// StrictMatch is ExactMatch that keeps letter case and inner punctuation
// (only spacing, curly quotes and a final full stop, ! or ? are ignored):
// the checkpoint's no-AI floor, so "my name is budi" never passes for a
// question about capital letters.
func StrictMatch(answer string, expected []string) bool {
	norm := func(s string) string {
		s = strings.NewReplacer("’", "'", "‘", "'", "“", `"`, "”", `"`).Replace(strings.TrimSpace(tutor.FoldAccents(s)))
		s = strings.TrimRight(s, ".!? ")
		return strings.Join(strings.Fields(s), " ")
	}
	a := norm(answer)
	if a == "" {
		return false
	}
	for _, e := range expected {
		if norm(e) == a {
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
