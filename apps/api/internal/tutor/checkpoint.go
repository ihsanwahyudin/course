package tutor

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
)

// Checkpoint-only question types. Besides multiple choice and word order:
const (
	// QuestionText is an open answer, typed or spoken, graded against
	// accepted answers and a rubric.
	QuestionText = "text"
	// QuestionLetters: spell a word by tapping letter tiles.
	QuestionLetters = "letters"
	// QuestionMulti: tick every right option; more than one is right.
	QuestionMulti = "multi"
	// QuestionMatch: pair each item on the left with one on the right.
	QuestionMatch = "match"
)

const (
	MinCheckItems      = 5
	MaxCheckItems      = 8
	MinLevelCheckItems = 10
	MaxLevelCheckItems = 15
	// A level checkpoint's questions come from at least two thirds of the
	// level's lessons, or from this many when the level is bigger.
	MaxLevelSpread = 10
	maxAccepted    = 20
	// A checkpoint question may quote a short text to read (an email, a
	// notice), so its prompt can be longer than a lesson step's.
	maxCheckPrompt = 800
	maxSpelled     = 15 // letters: the longest word to spell
	maxDecoys      = 3  // letters: extra tiles that aren't in the word
	maxMatchPairs  = 6
)

// CheckItem is one checkpoint question. Multiple choice and word order use
// the lesson question fields; letters uses Tokens (the tiles) and Solution
// (the word); multi uses Options and Answers; match uses Left, Right and
// Pairing; open answers use Accepted and Rubric. The answer key never
// leaves the server before grading.
type CheckItem struct {
	ID string `json:"id"`
	// LessonID: in a level checkpoint, the lesson the question comes from.
	LessonID string `json:"lessonId,omitempty"`
	Question
	// Answers: multi, the indexes of every right option.
	Answers []int `json:"answers,omitempty"`
	// Left and Right: match, the two columns. Pairing[i] is the index in
	// Right that goes with Left[i].
	Left     []string `json:"left,omitempty"`
	Right    []string `json:"right,omitempty"`
	Pairing  []int    `json:"pairing,omitempty"`
	Accepted []string `json:"acceptedAnswers"`
	Rubric   string   `json:"rubric"`
}

// Key is the item's correct answer as the learner would write it.
func (it CheckItem) Key() string {
	switch it.Type {
	case QuestionMCQ:
		if it.Answer >= 0 && it.Answer < len(it.Options) {
			return it.OptionsForDisplay()[it.Answer]
		}
	case QuestionWordOrder:
		return it.Solution
	case QuestionLetters:
		return spellOut(it.Solution)
	case QuestionMulti:
		return it.optionsText(it.Answers)
	case QuestionMatch:
		return it.pairsText(it.Pairing)
	}
	if len(it.Accepted) > 0 {
		return it.Accepted[0]
	}
	return ""
}

// DraftItem is a checkpoint question as the writer sends it, with the right
// options given by their text (models get an index wrong far more often)
// and a match question as its pairs.
type DraftItem struct {
	CheckItem
	CorrectOption  string      `json:"correctOption"`
	CorrectOptions []string    `json:"correctOptions"`
	Pairs          []MatchPair `json:"pairs"`
}

// MatchPair is one pair of a match question, as the writer gives it.
type MatchPair struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// FromDrafts turns a writer's questions into items: right options become
// indexes, and options and match columns are shuffled so the answer isn't
// always in the same place.
func FromDrafts(ds []DraftItem) ([]CheckItem, error) {
	items := make([]CheckItem, 0, len(ds))
	for i, d := range ds {
		it := d.CheckItem
		switch it.Type {
		case QuestionMCQ:
			it.Answer = optionIndex(it.Options, d.CorrectOption)
			if it.Answer < 0 {
				return nil, fmt.Errorf("question %d: the correct option %q isn't one of its options", i+1, d.CorrectOption)
			}
			order := shuffled(len(it.Options), it.Prompt)
			it.Options, it.Answer = permute(it.Options, order), order[it.Answer]
			if len(it.DisplayOptions) == len(order) {
				it.DisplayOptions = permute(it.DisplayOptions, order)
			}
		case QuestionMulti:
			it.Answers = nil
			for _, c := range d.CorrectOptions {
				j := optionIndex(it.Options, c)
				if j < 0 {
					return nil, fmt.Errorf("question %d: the correct option %q isn't one of its options", i+1, c)
				}
				it.Answers = append(it.Answers, j)
			}
			order := shuffled(len(it.Options), it.Prompt)
			it.Options = permute(it.Options, order)
			for k, a := range it.Answers {
				it.Answers[k] = order[a]
			}
			slices.Sort(it.Answers)
		case QuestionMatch:
			it.Left, it.Right, it.Pairing = nil, nil, nil
			for _, p := range d.Pairs {
				it.Left = append(it.Left, p.Left)
				it.Right = append(it.Right, p.Right)
			}
			order := shuffled(len(it.Right), it.Prompt)
			it.Right = permute(it.Right, order)
			it.Pairing = order
		}
		items = append(items, it)
	}
	return items, nil
}

// optionIndex finds an option by its text, or -1.
func optionIndex(options []string, text string) int {
	if strings.TrimSpace(text) == "" {
		return -1
	}
	for j, o := range options {
		if NormalizeWords(o) == NormalizeWords(text) {
			return j
		}
	}
	return -1
}

// shuffled is a permutation of n chosen by a hash of seed (stable for a
// question, different across questions): item i moves to order[i].
func shuffled(n int, seed string) []int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	r := rand.New(rand.NewPCG(uint64(h.Sum32()), 0x9e3779b97f4a7c15))
	return r.Perm(n)
}

func permute[T any](xs []T, order []int) []T {
	out := make([]T, len(xs))
	for from, to := range order {
		out[to] = xs[from]
	}
	return out
}

// ValidateCheckpoint checks a written checkpoint: 5–8 valid questions that
// together cover every objective and repeat none of the earlier ones. It
// numbers the items and shuffles word-order tokens that came in order.
func ValidateCheckpoint(items []CheckItem, objectives map[string]bool, earlier []string) ([]CheckItem, error) {
	return validateItems(items, objectives, earlier, MinCheckItems, MaxCheckItems)
}

// ValidateLevelCheckpoint checks a level checkpoint: 10–15 questions, each
// from one of the level's lessons and counted towards that lesson's
// objective, spread over at least LevelSpread of the lessons.
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
	if need := LevelSpread(len(lessons)); len(used) < need {
		return nil, fmt.Errorf("questions come from %d lessons; spread them over at least %d", len(used), need)
	}
	return validateItems(items, objectives, earlier, MinLevelCheckItems, MaxLevelCheckItems)
}

// LevelSpread is how many different lessons a level checkpoint must draw on.
func LevelSpread(lessons int) int {
	return min((2*lessons+2)/3, MaxLevelSpread)
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
	var repeats []int
	for i, it := range items {
		var err error
		switch it.Type {
		case QuestionText:
			it, err = validateTextItem(it, objectives)
		case QuestionMCQ, QuestionWordOrder:
			// The grader writes each explanation; a key explanation is optional here.
			if strings.TrimSpace(it.Explanation) == "" {
				it.Explanation = "-"
			}
			it.Question, err = validateQuestion(it.Question, objectives, maxCheckPrompt)
			it.Answers, it.Left, it.Right, it.Pairing = nil, nil, nil, nil
		default:
			it, err = validateTileItem(it, objectives)
		}
		if err == nil && it.Type != QuestionText {
			it.Accepted, it.Rubric = []string{it.Key()}, clean(it.Rubric)
		}
		if err != nil {
			return nil, fmt.Errorf("question %d: %w", i+1, err)
		}
		// A repeated question is dropped; the checkpoint fails only if too
		// few new ones are left (narrow lessons make some repeats likely).
		key := NormalizeWords(it.Prompt + " " + it.Listen)
		if seen[key] {
			repeats = append(repeats, i+1)
			continue
		}
		seen[key] = true
		if it.Type == QuestionWordOrder && !realWords(it.Tokens) {
			return nil, fmt.Errorf("question %d: word order is for the words of a sentence, not letters", i+1)
		}
		if it.Type == QuestionWordOrder && NormalizeWords(strings.Join(it.Tokens, " ")) == NormalizeWords(it.Solution) {
			it.Tokens = append(it.Tokens[1:], it.Tokens[0])
		}
		it.ID = "q" + strconv.Itoa(len(out)+1)
		covered[it.ObjectiveID] = true
		out = append(out, it)
	}
	if len(out) < min {
		return nil, fmt.Errorf("questions %v repeat earlier questions, leaving %d new ones; write at least %d new questions", repeats, len(out), min)
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
	if badLen(it.Prompt, maxCheckPrompt) || badLen(it.Rubric, maxText) {
		return it, fmt.Errorf("an open question needs a prompt and a rubric")
	}
	if !objectives[it.ObjectiveID] {
		return it, fmt.Errorf("question targets unknown objective %q", it.ObjectiveID)
	}
	if runes(it.Listen) > 300 {
		return it, fmt.Errorf("listen text must be at most 300 characters")
	}
	if it.Listen == "" && contextlessLetterRequest.MatchString(NormalizeWords(it.Prompt)) {
		return it, fmt.Errorf("a letter question needs a clue in its prompt or listening audio; an answer-length instruction alone is not a question")
	}
	if showsHeard(it.Prompt, it.Listen) {
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
	if err := validateSoundComparison(it.Prompt, acc); err != nil {
		return it, err
	}
	it.Accepted = acc
	it.Options, it.Answer, it.Tokens, it.Solution = nil, 0, nil, ""
	return it, nil
}

// validateTileItem checks a letters, multi or match question.
func validateTileItem(it CheckItem, objectives map[string]bool) (CheckItem, error) {
	if strings.TrimSpace(it.Explanation) == "" {
		it.Explanation = "-"
	}
	q, err := checkBasics(it.Question, objectives, maxCheckPrompt)
	if err != nil {
		return it, err
	}
	it.Question = q
	it.Options, it.Left, it.Right, it.Solution, it.Answer = cleanAll(it.Options), cleanAll(it.Left), cleanAll(it.Right), clean(it.Solution), 0
	switch it.Type {
	case QuestionLetters:
		return validateLetters(it)
	case QuestionMulti:
		return validateMulti(it)
	case QuestionMatch:
		return validateMatch(it)
	}
	return it, fmt.Errorf("unknown question type %q", it.Type)
}

// validateLetters: Solution is one word to spell; Tokens are its letters,
// shuffled, plus at most maxDecoys others. Without tokens the tiles are the
// word's own letters.
func validateLetters(it CheckItem) (CheckItem, error) {
	word := letters(it.Solution)
	if word == "" || word != strings.ToLower(FoldAccents(it.Solution)) || len(word) < 2 || len(word) > maxSpelled {
		return it, fmt.Errorf("a letters question spells one word of 2–%d letters", maxSpelled)
	}
	shown := strings.FieldsFunc(strings.ToLower(FoldAccents(it.Prompt)), func(r rune) bool { return r < 'a' || r > 'z' })
	if slices.Contains(shown, word) {
		return it, fmt.Errorf("a letters question must not show the word; play it (listen) or give its meaning")
	}
	tiles := make([]string, 0, len(word)+maxDecoys)
	for _, t := range it.Tokens {
		l := letters(t)
		if len(l) != 1 || runes(clean(t)) != 1 {
			return it, fmt.Errorf("letter tiles must be single letters")
		}
		tiles = append(tiles, strings.ToUpper(l))
	}
	if len(tiles) == 0 {
		for _, r := range word {
			tiles = append(tiles, strings.ToUpper(string(r)))
		}
	}
	if !fitsTiles(word, tiles) {
		return it, fmt.Errorf("the letter tiles must include every letter of %q", it.Solution)
	}
	if len(tiles) > len(word)+maxDecoys {
		return it, fmt.Errorf("at most %d extra letter tiles", maxDecoys)
	}
	if letters(strings.Join(tiles, "")) == word || len(it.Tokens) == 0 {
		tiles = permute(tiles, shuffled(len(tiles), it.Prompt+it.Solution))
		if letters(strings.Join(tiles, "")) == word {
			tiles = append(tiles[1:], tiles[0])
		}
	}
	it.Tokens, it.Solution = tiles, word
	it.Options, it.Answers, it.Left, it.Right, it.Pairing = nil, nil, nil, nil, nil
	return it, nil
}

// validateMulti: 3–6 distinct options, at least one right and one wrong.
func validateMulti(it CheckItem) (CheckItem, error) {
	if len(it.Options) < 3 || len(it.Options) > 6 || !distinct(it.Options) {
		return it, fmt.Errorf("a multi question needs 3–6 distinct options")
	}
	seen := map[int]bool{}
	for _, a := range it.Answers {
		if a < 0 || a >= len(it.Options) || seen[a] {
			return it, fmt.Errorf("multi answers must be distinct options")
		}
		seen[a] = true
	}
	if len(seen) == 0 || len(seen) == len(it.Options) {
		return it, fmt.Errorf("a multi question needs at least one right and one wrong option")
	}
	it.Answers = slices.Sorted(maps.Keys(seen))
	var keys []string
	for _, a := range it.Answers {
		keys = append(keys, it.Options[a])
	}
	if err := validateSoundComparison(it.Prompt, keys); err != nil {
		return it, err
	}
	it.Tokens, it.Solution, it.Left, it.Right, it.Pairing = nil, "", nil, nil, nil
	return it, nil
}

// validateMatch: 2–6 pairs, distinct on each side; the right column never
// starts in the answer's order.
func validateMatch(it CheckItem) (CheckItem, error) {
	n := len(it.Left)
	if n < 2 || n > maxMatchPairs || len(it.Right) != n || len(it.Pairing) != n {
		return it, fmt.Errorf("a match question needs 2–%d pairs", maxMatchPairs)
	}
	if !distinct(it.Left) || !distinct(it.Right) {
		return it, fmt.Errorf("match items must be distinct and short")
	}
	if !isPermutation(it.Pairing) {
		return it, fmt.Errorf("match pairing must use each right item once")
	}
	inOrder := true
	for i, p := range it.Pairing {
		inOrder = inOrder && p == i
	}
	if inOrder {
		it.Right = append(it.Right[1:], it.Right[0])
		for i := range it.Pairing {
			it.Pairing[i] = (i + n - 1) % n
		}
	}
	it.Options, it.Answers, it.Tokens, it.Solution = nil, nil, nil, ""
	return it, nil
}

func cleanAll(xs []string) []string {
	if xs == nil {
		return nil
	}
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = clean(x)
	}
	return out
}

// distinct: every entry is non-empty, short and different (ignoring case).
func distinct(xs []string) bool {
	seen := map[string]bool{}
	for _, x := range xs {
		k := strings.ToLower(x)
		if badLen(x, maxShort) || seen[k] {
			return false
		}
		seen[k] = true
	}
	return true
}

func isPermutation(xs []int) bool {
	seen := make([]bool, len(xs))
	for _, x := range xs {
		if x < 0 || x >= len(xs) || seen[x] {
			return false
		}
		seen[x] = true
	}
	return true
}

// letters keeps only the letters of s, lowercased: "H-O-U-S-E" → "house".
func letters(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(FoldAccents(s)) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// spellOut writes a word as its letters: "house" → "H-O-U-S-E".
func spellOut(word string) string {
	return strings.Join(strings.Split(strings.ToUpper(letters(word)), ""), "-")
}

// fitsTiles: s can be spelled with the tiles, each used at most once.
func fitsTiles(s string, tiles []string) bool {
	left := map[rune]int{}
	for _, t := range tiles {
		for _, r := range letters(t) {
			left[r]++
		}
	}
	for _, r := range s {
		if left[r] == 0 {
			return false
		}
		left[r]--
	}
	return true
}

// indexes parses a "2,0,1" answer: distinct indexes below n.
func indexes(answer string, n int) ([]int, bool) {
	var out []int
	seen := map[int]bool{}
	for _, f := range strings.Split(answer, ",") {
		i, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || i < 0 || i >= n || seen[i] {
			return nil, false
		}
		seen[i] = true
		out = append(out, i)
	}
	return out, len(out) > 0
}

func (it CheckItem) optionsText(picks []int) string {
	var out []string
	for _, i := range slices.Sorted(slices.Values(picks)) {
		if i >= 0 && i < len(it.Options) {
			out = append(out, it.Options[i])
		}
	}
	return strings.Join(out, ", ")
}

func (it CheckItem) pairsText(pairing []int) string {
	var out []string
	for i, j := range pairing {
		if i < len(it.Left) && j >= 0 && j < len(it.Right) {
			out = append(out, it.Left[i]+" → "+it.Right[j])
		}
	}
	return strings.Join(out, "; ")
}

// AnswerFits reports whether an answer has the question's shape: an option
// index for multiple choice, the given words for word order, letters from
// the tiles for letters, option indexes ("0,2") for multi, and one right
// item per left item ("2,0,1") for match.
func AnswerFits(it CheckItem, answer string) bool {
	switch it.Type {
	case QuestionMCQ:
		i, err := strconv.Atoi(strings.TrimSpace(answer))
		return err == nil && i >= 0 && i < len(it.Options)
	case QuestionWordOrder:
		return sameWords(it.Tokens, answer)
	case QuestionLetters:
		l := letters(answer)
		return l != "" && fitsTiles(l, it.Tokens)
	case QuestionMulti:
		_, ok := indexes(answer, len(it.Options))
		return ok
	case QuestionMatch:
		picks, ok := indexes(answer, len(it.Right))
		return ok && len(picks) == len(it.Left)
	}
	return true
}

// FixedVerdict grades what needs no judgement: multiple choice and word
// order are right or wrong; letters, multi and match earn part marks; an
// open answer equal to an accepted one is right. ok is false when an open
// answer needs the grader.
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
	case QuestionLetters:
		got, want := letters(answer), letters(it.Solution)
		switch {
		case got == want:
			return 100, true
		case len(want) >= 4 && editDistance(got, want) == 1:
			return 50, true // one letter wrong, missing or extra
		}
		return 0, true
	case QuestionMulti:
		picks, _ := indexes(answer, len(it.Options))
		right := 0
		for _, p := range picks {
			if slices.Contains(it.Answers, p) {
				right++
			}
		}
		// A wrong tick cancels a right one, so ticking everything earns nothing.
		net := max(0, right-(len(picks)-right))
		return int(math.Round(float64(net) * 100 / float64(max(1, len(it.Answers))))), true
	case QuestionMatch:
		picks, _ := indexes(answer, len(it.Right))
		right := 0
		for i, p := range picks {
			if i < len(it.Pairing) && it.Pairing[i] == p {
				right++
			}
		}
		return int(math.Round(float64(right) * 100 / float64(max(1, len(it.Pairing))))), true
	}
	if FormatSensitiveText(it) {
		return 0, false
	}
	// Legacy checkpoints may ask for a list of letter names in a text box.
	// "a dan e" and "E, A" have the same content as "a e"; this narrowly
	// scoped comparison never strips conjunctions from ordinary sentences.
	if (listText.MatchString(it.Prompt) && LetterListMatch(answer, it.Accepted)) || exact(answer, it.Accepted) {
		return 100, true
	}
	return 0, false
}

// AnswerText is an answer as a person reads it: the option's text for
// multiple choice and multi, spelled letters, the pairs for match.
func AnswerText(it CheckItem, answer string) string {
	answer = strings.TrimSpace(answer)
	switch it.Type {
	case QuestionMCQ:
		if n, err := strconv.Atoi(answer); err == nil && n >= 0 && n < len(it.Options) {
			return it.OptionsForDisplay()[n]
		}
	case QuestionLetters:
		return spellOut(answer)
	case QuestionMulti:
		if picks, ok := indexes(answer, len(it.Options)); ok {
			return it.optionsText(picks)
		}
	case QuestionMatch:
		if picks, ok := indexes(answer, len(it.Right)); ok {
			return it.pairsText(picks)
		}
	}
	return answer
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
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
func ApplyGrade(raw json.RawMessage, items []CheckItem, answers map[string]string, fixed map[string]int, locale ...string) (Review, map[string]int, string, error) {
	lang := "en"
	if len(locale) > 0 {
		lang = locale[0]
	}
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
	correctedVerdict := false
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
		// Out-of-range scores mean broken output, even where the server's score stands.
		if math.IsNaN(gi.Score) || gi.Score < 0 || gi.Score > 100 {
			return Review{}, nil, "", fmt.Errorf("%w: question %s score out of range", ErrMalformed, it.ID)
		}
		score, isFixed := fixed[it.ID]
		// A wrong key must not cost the learner: the grader's judgement
		// against the real answer stands instead.
		keyFixed := isFixed && gi.KeyWrong && clean(gi.CorrectAnswer) != "" && NormalizeWords(gi.CorrectAnswer) != NormalizeWords(it.Key())
		if !isFixed || keyFixed {
			score = int(math.Round(gi.Score))
		}
		correct := it.Key()
		if ((it.Type == QuestionText && !isFixed) || keyFixed) && clean(gi.CorrectAnswer) != "" {
			correct = clean(gi.CorrectAnswer)
		}
		ans := AnswerText(it, answers[it.ID])
		// A model cannot contradict the server's verdict or invent a case
		// requirement for an answer that is already known to be correct.
		caseOnly := it.Type == QuestionText && score == 100 && ans != it.Key() && strings.EqualFold(strings.TrimSpace(ans), strings.TrimSpace(it.Key()))
		if isFixed && !keyFixed && (int(math.Round(gi.Score)) != score || caseOnly) {
			expl = fixedGradeExplanation(score, correct, lang, caseOnly)
			correctedVerdict = true
		}
		rev.Items = append(rev.Items, ItemReview{
			ID: it.ID, LessonID: it.LessonID, ObjectiveID: it.ObjectiveID, Type: it.Type, Prompt: it.Prompt, Options: it.OptionsForDisplay(), Listen: it.Listen,
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
	feedback := clean(g.Feedback)
	if correctedVerdict {
		// Feedback and review advice from the same contradicted verdict may
		// repeat its invented requirement. Build them from final scores.
		rev.ToReview = []string{}
		for i, it := range rev.Items {
			if it.Score < 100 && len(rev.ToReview) < 4 {
				label := fmt.Sprintf("Question %d", i+1)
				if lang == "id" {
					label = fmt.Sprintf("Soal %d", i+1)
				}
				rev.ToReview = append(rev.ToReview, label)
			}
		}
		if lang == "id" {
			feedback = "Lihat ulasan tiap jawaban untuk bagian yang perlu dilatih."
			if len(rev.ToReview) == 0 {
				feedback = "Semua jawaban benar. Kerja bagus!"
			}
		} else {
			feedback = "See each answer's review for what to practise."
			if len(rev.ToReview) == 0 {
				feedback = "Every answer is correct. Well done!"
			}
		}
	}
	return rev, scores, feedback, nil
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
