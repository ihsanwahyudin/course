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
		r.Message = say(in, "welcome") + objectiveList(in.Node) + ". " + enterPhase(in, l, tutor.PhaseExplanation)
		r.Action = string(tutor.ActionAdvancePhase)

	case tutor.PhaseExplanation:
		if t < len(l.KeyPoints) {
			r.Message = say(in, "next_idea") + keyPoint(in, l.KeyPoints[t])
		} else {
			r.Message = say(in, "main_ideas") + enterPhase(in, l, tutor.PhaseGuidedPractice)
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
		// Work out which prompt the learner is answering from the tutor's
		// previous message, so a wrong answer is re-asked rather than skipped.
		idx := currentPrompt(in.Messages, prompts)
		if idx < 0 {
			r.Message = say(in, "lets_practise") + prompts[0].Prompt
			break
		}
		answered := prompts[idx]
		if !Matches(last, answered.Expected) {
			r.Message = fmt.Sprintf(say(in, "not_quite"), answered.Hint, answered.Prompt)
			r.Action = string(tutor.ActionAskRetry)
			r.ObservedMistakes = []string{answered.ObjectiveID}
			break
		}
		if idx+1 >= len(prompts) && t >= 2 {
			r.Message = say(in, "correct") + enterPhase(in, l, next)
			r.Action = string(tutor.ActionAdvancePhase)
		} else {
			r.Message = say(in, "correct") + say(in, "next") + prompts[(idx+1)%len(prompts)].Prompt
		}

	default:
		r.Message = say(in, "keep_going")
	}
	return json.Marshal(r)
}

// currentPrompt finds the practice prompt quoted in the tutor's last message.
func currentPrompt(msgs []ChatMessage, prompts []curriculum.PracticePrompt) int {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != "assistant" {
			continue
		}
		best, at := -1, -1
		for j, p := range prompts {
			if k := strings.LastIndex(msgs[i].Content, p.Prompt); k > at {
				best, at = j, k
			}
		}
		return best
	}
	return -1
}

// enterPhase is the bridge text the tutor says when moving into a phase.
func enterPhase(in TutorInput, l *curriculum.Lesson, p tutor.Phase) string {
	switch p {
	case tutor.PhaseExplanation:
		if len(l.KeyPoints) > 0 {
			return say(in, "first_idea") + keyPoint(in, l.KeyPoints[0])
		}
	case tutor.PhaseGuidedPractice:
		return say(in, "practise_together") + l.GuidedPractice[0].Prompt
	case tutor.PhaseIndependentPractice:
		return say(in, "on_your_own") + l.IndependentPrompt[0].Prompt
	case tutor.PhaseAssessment:
		return say(in, "ready_assessment")
	}
	return ""
}

func keyPoint(in TutorInput, k curriculum.KeyPoint) string {
	s := k.Title + ": " + k.Explanation
	if len(k.Examples) > 0 {
		s += fmt.Sprintf(say(in, "example"), k.Examples[0].EN, k.Examples[0].ID)
	}
	return s + say(in, "reply_ready")
}

// Mock phrasing per UI language. Lesson content (key points, prompts) stays
// as written in the course files.
var mockText = map[string]map[string]string{
	"en": {
		"welcome": "Great to have you here! In this lesson we will practise: ", "next_idea": "Good. Next idea — ",
		"main_ideas": "You have the main ideas. ", "lets_practise": "Let's practise. ", "not_quite": "Not quite. Hint: %s Try again: %s",
		"correct": "Correct, well done! ", "next": "Next: ", "keep_going": "Let's keep going.", "first_idea": "First idea — ",
		"practise_together": "Let's practise together. ", "on_your_own": "Now try on your own. ",
		"ready_assessment": "Great practice — you're ready for the assessment.", "example": " Example: %q (%s).",
		"reply_ready": " Reply when you're ready to continue.", "all_correct": "Excellent work — every answer met the objective.",
		"review": "Review and try again: %s.", "correct_answer": "Correct: %q", "none_correct": "No fully correct answers yet.",
	},
	"id": {
		"welcome": "Senang kamu di sini! Di pelajaran ini kita akan berlatih: ", "next_idea": "Bagus. Ide berikutnya — ",
		"main_ideas": "Kamu sudah paham ide utamanya. ", "lets_practise": "Ayo berlatih. ", "not_quite": "Belum tepat. Petunjuk: %s Coba lagi: %s",
		"correct": "Benar, bagus sekali! ", "next": "Berikutnya: ", "keep_going": "Ayo lanjutkan.", "first_idea": "Ide pertama — ",
		"practise_together": "Ayo berlatih bersama. ", "on_your_own": "Sekarang coba sendiri. ",
		"ready_assessment": "Latihan yang bagus — kamu siap untuk asesmen.", "example": " Contoh: %q (%s).",
		"reply_ready": " Balas kalau kamu siap lanjut.", "all_correct": "Kerja bagus — semua jawaban memenuhi tujuan pelajaran.",
		"review": "Ulas lagi lalu coba kembali: %s.", "correct_answer": "Benar: %q", "none_correct": "Belum ada jawaban yang sepenuhnya benar.",
	},
}

func say(in TutorInput, key string) string { return phrase(in.Locale, key) }

func phrase(locale, key string) string {
	if m, ok := mockText[locale]; ok {
		return m[key]
	}
	return mockText["en"][key]
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
