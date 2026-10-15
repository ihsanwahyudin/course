package tutor

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
)

// QuestionText is an open checkpoint answer, typed or spoken, graded against
// accepted answers and a rubric.
const QuestionText = "text"

const (
	MinCheckItems      = 5
	MaxCheckItems      = 8
	MinLevelCheckItems = 10
	MaxLevelCheckItems = 15
	maxAccepted        = 20
)

// CheckItem is one checkpoint question. Multiple choice and word order use
// the lesson question fields; open answers use Accepted and Rubric. The
// answer key never leaves the server before grading.
type CheckItem struct {
	ID string `json:"id"`
	// LessonID: in a level checkpoint, the lesson the question comes from.
	LessonID string `json:"lessonId,omitempty"`
	Question
	Accepted []string `json:"acceptedAnswers"`
	Rubric   string   `json:"rubric"`
}

// Key is the item's correct answer as the learner would write it.
func (it CheckItem) Key() string {
	switch it.Type {
	case QuestionMCQ:
		if it.Answer >= 0 && it.Answer < len(it.Options) {
			return it.Options[it.Answer]
		}
	case QuestionWordOrder:
		return it.Solution
	}
	if len(it.Accepted) > 0 {
		return it.Accepted[0]
	}
	return ""
}

// DraftItem is a checkpoint question as the writer sends it, with the right
// option given by its text (models get an index wrong far more often).
type DraftItem struct {
	CheckItem
	CorrectOption string `json:"correctOption"`
}

// FromDrafts turns a writer's questions into items: the right option becomes
// its index, and options are shuffled so the right one isn't always first.
func FromDrafts(ds []DraftItem) ([]CheckItem, error) {
	items := make([]CheckItem, 0, len(ds))
	for i, d := range ds {
		it := d.CheckItem
		if it.Type == QuestionMCQ {
			it.Answer = -1
			for j, o := range it.Options {
				if NormalizeWords(o) == NormalizeWords(d.CorrectOption) && strings.TrimSpace(d.CorrectOption) != "" {
					it.Answer = j
					break
				}
			}
			if it.Answer < 0 {
				return nil, fmt.Errorf("question %d: the correct option %q isn't one of its options", i+1, d.CorrectOption)
			}
			it.Options, it.Answer = shuffleOptions(it.Options, it.Answer, it.Prompt)
		}
		items = append(items, it)
	}
	return items, nil
}

// shuffleOptions reorders options by a hash of the prompt (stable for a
// question, different across questions).
func shuffleOptions(options []string, answer int, seed string) ([]string, int) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	r := rand.New(rand.NewPCG(uint64(h.Sum32()), 0x9e3779b97f4a7c15))
	order := r.Perm(len(options))
	out := make([]string, len(options))
	newAnswer := 0
	for to, from := range order {
		out[to] = options[from]
		if from == answer {
			newAnswer = to
		}
	}
	return out, newAnswer
}

// ValidateCheckpoint checks a written checkpoint: 5–8 valid questions that
// together cover every objective and repeat none of the earlier ones. It
// numbers the items and shuffles word-order tokens that came in order.
func ValidateCheckpoint(items []CheckItem, objectives map[string]bool, earlier []string) ([]CheckItem, error) {
	return validateItems(items, objectives, earlier, MinCheckItems, MaxCheckItems)
}

// ValidateLevelCheckpoint checks a level checkpoint: 10–15 questions, each
// from one of the level's lessons and counted towards that lesson's
// objective, spread over at least two thirds of the lessons.
// lessons maps each lesson id to its checkpoint objective.
func ValidateLevelCheckpoint(items []CheckItem, lessons map[string]string, earlier []string) ([]CheckItem, error) {
	objectives := map[string]bool{}
	for _, o := range lessons {
		objectives[o] = true
	}
	used := map[string]bool{}
	for i, it := range items {
		o, ok := lessons[it.LessonID]
		if !ok {
			return nil, fmt.Errorf("question %d names lesson %q, which this checkpoint doesn't cover", i+1, it.LessonID)
		}
		if it.ObjectiveID != o {
			return nil, fmt.Errorf("question %d: a question from %s counts towards %q, not %q", i+1, it.LessonID, o, it.ObjectiveID)
		}
		used[it.LessonID] = true
	}
	if need := (2*len(lessons) + 2) / 3; len(used) < need {
		return nil, fmt.Errorf("questions come from %d lessons; spread them over at least %d", len(used), need)
	}
	return validateItems(items, objectives, earlier, MinLevelCheckItems, MaxLevelCheckItems)
}

func validateItems(items []CheckItem, objectives map[string]bool, earlier []string, min, max int) ([]CheckItem, error) {
	if len(items) < min || len(items) > max {
		return nil, fmt.Errorf("a checkpoint needs %d–%d questions, got %d", min, max, len(items))
	}
	seen := map[string]bool{}
	for _, p := range earlier {
		seen[NormalizeWords(p)] = true
	}
	covered := map[string]bool{}
	out := make([]CheckItem, 0, len(items))
	for i, it := range items {
		var err error
		if it.Type == QuestionText {
			it, err = validateTextItem(it, objectives)
		} else {
			// The grader writes each explanation; a key explanation is optional here.
			if strings.TrimSpace(it.Explanation) == "" {
				it.Explanation = "-"
			}
			it.Question, err = ValidateQuestion(it.Question, objectives)
			it.Accepted, it.Rubric = []string{it.Key()}, clean(it.Rubric)
		}
		if err != nil {
			return nil, fmt.Errorf("question %d: %w", i+1, err)
		}
		key := NormalizeWords(it.Prompt + " " + it.Listen)
		if seen[key] {
			return nil, fmt.Errorf("question %d repeats an earlier question", i+1)
		}
		seen[key] = true
		if it.Type == QuestionWordOrder && !realWords(it.Tokens) {
			return nil, fmt.Errorf("question %d: word order is for the words of a sentence, not letters", i+1)
		}
		if it.Type == QuestionWordOrder && NormalizeWords(strings.Join(it.Tokens, " ")) == NormalizeWords(it.Solution) {
			it.Tokens = append(it.Tokens[1:], it.Tokens[0])
		}
		it.ID = "q" + strconv.Itoa(i+1)
		covered[it.ObjectiveID] = true
		out = append(out, it)
	}
	for id := range objectives {
		if !covered[id] {
			return nil, fmt.Errorf("no question tests objective %q", id)
		}
	}
	return out, nil
}

// realWords: most pieces are words (2+ letters), not single letters to spell.
func realWords(tokens []string) bool {
	words := 0
	for _, t := range tokens {
		if runes(NormalizeWords(t)) >= 2 {
			words++
		}
	}
	return len(tokens) >= 3 && words*2 >= len(tokens)
}

func validateTextItem(it CheckItem, objectives map[string]bool) (CheckItem, error) {
	it.Prompt, it.ObjectiveID, it.Rubric, it.Listen = clean(it.Prompt), clean(it.ObjectiveID), clean(it.Rubric), clean(it.Listen)
	it.Explanation = clean(it.Explanation)
	if badLen(it.Prompt, maxShort*2) || badLen(it.Rubric, maxText) {
		return it, fmt.Errorf("an open question needs a prompt and a rubric")
	}
	if !objectives[it.ObjectiveID] {
		return it, fmt.Errorf("question targets unknown objective %q", it.ObjectiveID)
	}
	if runes(it.Listen) > 300 {
		return it, fmt.Errorf("listen text must be at most 300 characters")
	}
	if it.Listen != "" && strings.Contains(NormalizeWords(it.Prompt), NormalizeWords(it.Listen)) {
		return it, fmt.Errorf("the prompt must not show what the learner should hear")
	}
	var acc []string
	for _, a := range it.Accepted {
		if a = clean(a); a != "" && runes(a) <= maxShort {
			acc = append(acc, a)
		}
	}
	if len(acc) == 0 || len(acc) > maxAccepted {
		return it, fmt.Errorf("an open question needs 1–%d accepted answers", maxAccepted)
	}
	it.Accepted = acc
	it.Options, it.Answer, it.Tokens, it.Solution = nil, 0, nil, ""
	return it, nil
}

// AnswerFits reports whether an answer has the question's shape: an option
// index for multiple choice, the given words for word order.
func AnswerFits(it CheckItem, answer string) bool {
	switch it.Type {
	case QuestionMCQ:
		i, err := strconv.Atoi(strings.TrimSpace(answer))
		return err == nil && i >= 0 && i < len(it.Options)
	case QuestionWordOrder:
		return sameWords(it.Tokens, answer)
	}
	return true
}

// FixedVerdict grades what needs no judgement: multiple choice and word
// order are right or wrong, and an open answer equal to an accepted one is
// right. ok is false when an open answer needs the grader.
func FixedVerdict(it CheckItem, answer string, exact func(string, []string) bool) (score int, ok bool) {
	switch it.Type {
	case QuestionMCQ:
		i, err := strconv.Atoi(strings.TrimSpace(answer))
		if err == nil && i == it.Answer {
			return 100, true
		}
		return 0, true
	case QuestionWordOrder:
		if NormalizeWords(answer) == NormalizeWords(it.Solution) {
			return 100, true
		}
		return 0, true
	}
	if exact(answer, it.Accepted) {
		return 100, true
	}
	return 0, false
}

// ItemReview is how one answer was graded, kept with the attempt.
type ItemReview struct {
	ID          string   `json:"id"`
	LessonID    string   `json:"lessonId,omitempty"`
	ObjectiveID string   `json:"objectiveId"`
	Type        string   `json:"type"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options,omitempty"`
	Listen      string   `json:"listen,omitempty"`
	// Answer is what the learner gave (an mcq answer as the option's text).
	Answer        string `json:"answer"`
	CorrectAnswer string `json:"correctAnswer"`
	Score         int    `json:"score"`
	Verdict       string `json:"verdict"` // correct | partial | wrong
	Explanation   string `json:"explanation"`
	// KeyFixed: the question's answer key was wrong; the grader judged the
	// answer against the real one.
	KeyFixed bool `json:"keyFixed,omitempty"`
}

// Review is the graded checkpoint: every answer plus the overall picture.
type Review struct {
	Items     []ItemReview `json:"items"`
	Strengths []string     `json:"strengths"`
	ToReview  []string     `json:"toReview"`
	// Lessons: in a level checkpoint, each lesson's score (mean of its questions).
	Lessons map[string]int `json:"lessons,omitempty"`
}

const (
	VerdictCorrect = "correct"
	VerdictPartial = "partial"
	VerdictWrong   = "wrong"
)

func verdict(score int) string {
	switch {
	case score >= 80:
		return VerdictCorrect
	case score > 0:
		return VerdictPartial
	}
	return VerdictWrong
}

// Grade is the grader's raw judgement.
type Grade struct {
	Items []struct {
		ID            string  `json:"id"`
		KeyWrong      bool    `json:"keyWrong"`
		Score         float64 `json:"score"`
		CorrectAnswer string  `json:"correctAnswer"`
		Explanation   string  `json:"explanation"`
	} `json:"items"`
	Strengths []string `json:"strengths"`
	ToReview  []string `json:"toReview"`
	Feedback  string   `json:"feedback"`
}

// ApplyGrade merges the grader's judgement with the fixed verdicts: fixed
// items keep their score whatever the grader says, every item needs an
// explanation, and open answers need a score in 0–100. It returns the
// review, per-objective scores (mean of their items) and the feedback.
func ApplyGrade(raw json.RawMessage, items []CheckItem, answers map[string]string, fixed map[string]int) (Review, map[string]int, string, error) {
	var g Grade
	if err := json.Unmarshal(raw, &g); err != nil {
		return Review{}, nil, "", fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	byID := map[string]int{}
	for i, gi := range g.Items {
		byID[gi.ID] = i
	}
	rev := Review{Items: make([]ItemReview, 0, len(items)), Strengths: cleanList(g.Strengths, 4), ToReview: cleanList(g.ToReview, 4)}
	sum, count := map[string]int{}, map[string]int{}
	for _, it := range items {
		i, ok := byID[it.ID]
		if !ok {
			return Review{}, nil, "", fmt.Errorf("%w: no grade for question %s", ErrMalformed, it.ID)
		}
		gi := g.Items[i]
		expl := clean(gi.Explanation)
		if badLen(expl, maxText) {
			return Review{}, nil, "", fmt.Errorf("%w: question %s needs an explanation", ErrMalformed, it.ID)
		}
		score, isFixed := fixed[it.ID]
		// A wrong key must not cost the learner: the grader's judgement
		// against the real answer stands instead.
		keyFixed := isFixed && gi.KeyWrong && clean(gi.CorrectAnswer) != ""
		if !isFixed || keyFixed {
			if math.IsNaN(gi.Score) || gi.Score < 0 || gi.Score > 100 {
				return Review{}, nil, "", fmt.Errorf("%w: question %s score out of range", ErrMalformed, it.ID)
			}
			score = int(math.Round(gi.Score))
		}
		correct := it.Key()
		if (it.Type == QuestionText || keyFixed) && clean(gi.CorrectAnswer) != "" {
			correct = clean(gi.CorrectAnswer)
		}
		ans := strings.TrimSpace(answers[it.ID])
		if it.Type == QuestionMCQ {
			if n, err := strconv.Atoi(ans); err == nil && n >= 0 && n < len(it.Options) {
				ans = it.Options[n]
			}
		}
		rev.Items = append(rev.Items, ItemReview{
			ID: it.ID, LessonID: it.LessonID, ObjectiveID: it.ObjectiveID, Type: it.Type, Prompt: it.Prompt, Options: it.Options, Listen: it.Listen,
			Answer: ans, CorrectAnswer: correct, Score: score, Verdict: verdict(score), Explanation: expl, KeyFixed: keyFixed,
		})
		sum[it.ObjectiveID] += score
		count[it.ObjectiveID]++
	}
	scores := map[string]int{}
	for id, n := range count {
		scores[id] = int(math.Round(float64(sum[id]) / float64(n)))
	}
	lsum, lcount := map[string]int{}, map[string]int{}
	for _, r := range rev.Items {
		if r.LessonID != "" {
			lsum[r.LessonID] += r.Score
			lcount[r.LessonID]++
		}
	}
	if len(lcount) > 0 {
		rev.Lessons = map[string]int{}
		for id, n := range lcount {
			rev.Lessons[id] = int(math.Round(float64(lsum[id]) / float64(n)))
		}
	}
	return rev, scores, clean(g.Feedback), nil
}

func cleanList(in []string, max int) []string {
	out := []string{}
	for _, s := range in {
		if s = clean(s); s != "" && runes(s) <= maxShort && len(out) < max {
			out = append(out, s)
		}
	}
	return out
}
