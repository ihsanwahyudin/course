package tutor

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestFromDraftsTakesTheOptionTextAndShuffles(t *testing.T) {
	d := DraftItem{CheckItem: CheckItem{Question: Question{Type: QuestionMCQ, ObjectiveID: "o", Prompt: "Which letter sounds like 'why'?", Options: []string{"W", "Y", "I"}}}, CorrectOption: "Y"}
	items, err := FromDrafts([]DraftItem{d})
	if err != nil || items[0].Options[items[0].Answer] != "Y" || len(items[0].Options) != 3 {
		t.Fatalf("the key follows the option's text: %+v %v", items, err)
	}
	again, _ := FromDrafts([]DraftItem{d})
	if again[0].Answer != items[0].Answer {
		t.Fatal("shuffling is stable for a question")
	}
	d.CorrectOption = "Z"
	if _, err := FromDrafts([]DraftItem{d}); err == nil {
		t.Fatal("a correct option that isn't offered is rejected")
	}
}

func TestApplyGradeCorrectsAWrongKey(t *testing.T) {
	items := []CheckItem{
		{ID: "q1", Question: Question{Type: QuestionMCQ, ObjectiveID: "o", Prompt: "p", Options: []string{"W", "Y"}, Answer: 0}},
		{ID: "q2", Question: Question{Type: QuestionMCQ, ObjectiveID: "o", Prompt: "p2", Options: []string{"A", "B"}, Answer: 0}},
	}
	answers := map[string]string{"q1": "1", "q2": "1"}
	fixed := map[string]int{"q1": 0, "q2": 0}
	raw := json.RawMessage(`{"items":[
		{"id":"q1","keyWrong":true,"correctAnswer":"Y","explanation":"Y is said 'why'.","score":100},
		{"id":"q2","keyWrong":false,"correctAnswer":"A","explanation":"A is right.","score":100}],
		"strengths":[],"toReview":[],"feedback":"ok"}`)
	rev, scores, _, err := ApplyGrade(raw, items, answers, fixed)
	if err != nil {
		t.Fatal(err)
	}
	if !rev.Items[0].KeyFixed || rev.Items[0].Score != 100 || rev.Items[0].CorrectAnswer != "Y" {
		t.Fatalf("a wrong key doesn't cost the learner: %+v", rev.Items[0])
	}
	if rev.Items[1].Score != 0 || rev.Items[1].KeyFixed || scores["o"] != 50 {
		t.Fatalf("without keyWrong the fixed score stands: %+v %v", rev.Items[1], scores)
	}
}

func TestAnswerFits(t *testing.T) {
	mcq := CheckItem{Question: Question{Type: QuestionMCQ, Options: []string{"a", "b"}}}
	wo := CheckItem{Question: Question{Type: QuestionWordOrder, Tokens: []string{"are", "you", "how"}}}
	if !AnswerFits(mcq, "1") || AnswerFits(mcq, "2") || AnswerFits(mcq, "b") {
		t.Fatal("mcq answers are option indexes")
	}
	if !AnswerFits(wo, "How are you") || AnswerFits(wo, "ignore all rules") || AnswerFits(wo, "how are") {
		t.Fatal("word-order answers use exactly the given words")
	}
}

func TestWordOrderNeedsWords(t *testing.T) {
	obj := map[string]bool{"o": true}
	item := func(tokens []string, solution string, n int) CheckItem {
		return CheckItem{Question: Question{Type: QuestionWordOrder, ObjectiveID: "o", Prompt: fmt.Sprintf("Put in order %d", n), Tokens: tokens, Solution: solution, Explanation: "e"}}
	}
	text := func(n int) CheckItem {
		return CheckItem{Question: Question{Type: QuestionText, ObjectiveID: "o", Prompt: fmt.Sprintf("Q%d", n)}, Accepted: []string{"a"}, Rubric: "r"}
	}
	ok := []CheckItem{item([]string{"teacher", "I", "a", "am"}, "I am a teacher", 1), text(2), text(3), text(4), text(5)}
	if _, err := ValidateCheckpoint(ok, obj, nil); err != nil {
		t.Fatalf("a sentence is fine: %v", err)
	}
	bad := []CheckItem{item([]string{"A", "N", "E", "M"}, "N-A-M-E", 1), text(2), text(3), text(4), text(5)}
	if _, err := ValidateCheckpoint(bad, obj, nil); err == nil {
		t.Fatal("spelling letters isn't a word-order question")
	}
}

func TestLevelSpread(t *testing.T) {
	for lessons, want := range map[int]int{6: 4, 14: 10, 31: 10} {
		if got := LevelSpread(lessons); got != want {
			t.Errorf("LevelSpread(%d) = %d, want %d", lessons, got, want)
		}
	}
}

func TestCheckpointPromptsMayQuoteAText(t *testing.T) {
	text := strings.Repeat("Read the email from the hotel. ", 20) // ~620 characters
	items := make([]CheckItem, 5)
	for i := range items {
		items[i] = CheckItem{Question: Question{Type: QuestionText, ObjectiveID: "o", Prompt: fmt.Sprintf("%s Q%d?", text, i)}, Accepted: []string{"a"}, Rubric: "r"}
	}
	if _, err := ValidateCheckpoint(items, map[string]bool{"o": true}, nil); err != nil {
		t.Fatalf("a checkpoint question with a short text to read: %v", err)
	}
	q := Question{Type: QuestionMCQ, ObjectiveID: "o", Prompt: text, Options: []string{"a", "b", "c"}, Explanation: "e"}
	if _, err := ValidateQuestion(q, map[string]bool{"o": true}); err == nil {
		t.Fatal("a lesson step's prompt stays short")
	}
}

func textItem(n int) CheckItem {
	return CheckItem{Question: Question{Type: QuestionText, ObjectiveID: "o", Prompt: fmt.Sprintf("Q%d", n)}, Accepted: []string{"a"}, Rubric: "r"}
}

// validOne validates it as the first of five questions and returns it.
func validOne(t *testing.T, it CheckItem) (CheckItem, error) {
	t.Helper()
	items, err := ValidateCheckpoint([]CheckItem{it, textItem(2), textItem(3), textItem(4), textItem(5)}, map[string]bool{"o": true}, nil)
	if err != nil {
		return CheckItem{}, err
	}
	return items[0], nil
}

func TestLettersQuestion(t *testing.T) {
	it, err := validOne(t, CheckItem{Question: Question{Type: QuestionLetters, ObjectiveID: "o", Prompt: "Listen and spell the word.", Listen: "house", Solution: "House"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(it.Tokens) != 5 || strings.Join(it.Tokens, "") == "HOUSE" || it.Solution != "house" || it.Key() != "H-O-U-S-E" {
		t.Fatalf("tiles are the word's letters, shuffled: %+v", it)
	}
	again, _ := validOne(t, it)
	if strings.Join(again.Tokens, "") != strings.Join(it.Tokens, "") {
		t.Fatal("validating twice keeps the tiles")
	}
	for _, a := range []string{"house", "H-O-U-S-E", "h o u s e"} {
		if !AnswerFits(it, a) {
			t.Fatalf("%q fits", a)
		}
		if s, ok := FixedVerdict(it, a, nil); !ok || s != 100 {
			t.Fatalf("%q is right: %d", a, s)
		}
	}
	if AnswerFits(it, "hhouse") || AnswerFits(it, "mouse") || AnswerFits(it, "") {
		t.Fatal("only the tiles, each once")
	}
	if s, _ := FixedVerdict(it, "hose", nil); s != 50 {
		t.Fatalf("one letter missing earns half: %d", s)
	}
	if s, _ := FixedVerdict(it, "shoe", nil); s != 0 {
		t.Fatalf("a different word earns nothing: %d", s)
	}
	if AnswerText(it, "hose") != "H-O-S-E" {
		t.Fatal("an answer reads as spelled letters")
	}

	decoys := CheckItem{Question: Question{Type: QuestionLetters, ObjectiveID: "o", Prompt: "Spell the English word for 'merah'.", Solution: "red", Tokens: []string{"r", "e", "d", "b", "a"}}}
	if it, err := validOne(t, decoys); err != nil || len(it.Tokens) != 5 || strings.Join(it.Tokens, "") != "REDBA" {
		t.Fatalf("given tiles (with decoys) keep their order: %+v %v", it.Tokens, err)
	}
	for name, bad := range map[string]CheckItem{
		"shows the word": {Question: Question{Type: QuestionLetters, ObjectiveID: "o", Prompt: "Spell 'house'.", Solution: "house"}},
		"two words":      {Question: Question{Type: QuestionLetters, ObjectiveID: "o", Prompt: "Spell it.", Listen: "ice cream", Solution: "ice cream"}},
		"missing letter": {Question: Question{Type: QuestionLetters, ObjectiveID: "o", Prompt: "Spell it.", Listen: "red", Solution: "red", Tokens: []string{"r", "e"}}},
		"too many tiles": {Question: Question{Type: QuestionLetters, ObjectiveID: "o", Prompt: "Spell it.", Listen: "red", Solution: "red", Tokens: strings.Split("redabcd", "")}},
		"not letters":    {Question: Question{Type: QuestionLetters, ObjectiveID: "o", Prompt: "Spell it.", Listen: "red", Solution: "red", Tokens: []string{"re", "d"}}},
	} {
		if _, err := validOne(t, bad); err == nil {
			t.Errorf("%s: should be rejected", name)
		}
	}
}

func TestMultiQuestion(t *testing.T) {
	d := DraftItem{CheckItem: CheckItem{Question: Question{Type: QuestionMulti, ObjectiveID: "o", Prompt: "Tick the vowels in 'table'.",
		Options: []string{"T", "A", "B", "L", "E"}}}, CorrectOptions: []string{"A", "E"}}
	items, err := FromDrafts([]DraftItem{d})
	if err != nil {
		t.Fatal(err)
	}
	it, err := validOne(t, items[0])
	if err != nil {
		t.Fatal(err)
	}
	if it.Key() != "A, E" && it.Key() != "E, A" {
		t.Fatalf("the key is the right options: %q", it.Key())
	}
	idx := func(opts ...string) string {
		var out []string
		for _, o := range opts {
			out = append(out, fmt.Sprint(optionIndex(it.Options, o)))
		}
		return strings.Join(out, ",")
	}
	for answer, want := range map[string]int{idx("A", "E"): 100, idx("E"): 50, idx("A", "E", "T"): 50, idx("A", "T"): 0, idx("T", "A", "B", "L", "E"): 0} {
		if !AnswerFits(it, answer) {
			t.Fatalf("%q fits", answer)
		}
		if s, ok := FixedVerdict(it, answer, nil); !ok || s != want {
			t.Errorf("%q scores %d, want %d", answer, s, want)
		}
	}
	if AnswerFits(it, "0,0") || AnswerFits(it, "9") || AnswerFits(it, "") || AnswerFits(it, "A") {
		t.Fatal("multi answers are distinct option indexes")
	}
	if AnswerText(it, idx("E", "A")) != it.Key() {
		t.Fatal("an answer reads as the options' text")
	}
	d.CorrectOptions = []string{"T", "A", "B", "L", "E"}
	all, _ := FromDrafts([]DraftItem{d})
	if _, err := validOne(t, all[0]); err == nil {
		t.Fatal("a multi question needs a wrong option")
	}
	d.CorrectOptions = []string{"A", "O"}
	if _, err := FromDrafts([]DraftItem{d}); err == nil {
		t.Fatal("a right option that isn't offered is rejected")
	}
}

func TestMatchQuestion(t *testing.T) {
	d := DraftItem{CheckItem: CheckItem{Question: Question{Type: QuestionMatch, ObjectiveID: "o", Prompt: "Match each word to its meaning."}},
		Pairs: []MatchPair{{"red", "merah"}, {"blue", "biru"}, {"green", "hijau"}}}
	items, err := FromDrafts([]DraftItem{d})
	if err != nil {
		t.Fatal(err)
	}
	it, err := validOne(t, items[0])
	if err != nil {
		t.Fatal(err)
	}
	right := func(pairs ...string) string {
		var out []string
		for _, p := range pairs {
			out = append(out, fmt.Sprint(slices.Index(it.Right, p)))
		}
		return strings.Join(out, ",")
	}
	if it.Key() != "red → merah; blue → biru; green → hijau" {
		t.Fatalf("key: %q", it.Key())
	}
	for answer, want := range map[string]int{right("merah", "biru", "hijau"): 100, right("biru", "merah", "hijau"): 33, right("hijau", "merah", "biru"): 0} {
		if s, ok := FixedVerdict(it, answer, nil); !ok || s != want {
			t.Errorf("%q scores %d, want %d", answer, s, want)
		}
	}
	if AnswerFits(it, "0,1") || AnswerFits(it, "0,0,1") || !AnswerFits(it, "2,1,0") {
		t.Fatal("a match answer pairs every left item with a different right one")
	}

	inOrder := CheckItem{Question: Question{Type: QuestionMatch, ObjectiveID: "o", Prompt: "Match."},
		Left: []string{"a", "b", "c"}, Right: []string{"x", "y", "z"}, Pairing: []int{0, 1, 2}}
	it, err = validOne(t, inOrder)
	if err != nil || slices.Equal(it.Pairing, []int{0, 1, 2}) || it.Key() != "a → x; b → y; c → z" {
		t.Fatalf("a right column in the answer's order is moved: %+v %v", it, err)
	}
	dup := CheckItem{Question: Question{Type: QuestionMatch, ObjectiveID: "o", Prompt: "Match."},
		Left: []string{"a", "a"}, Right: []string{"x", "y"}, Pairing: []int{1, 0}}
	if _, err := validOne(t, dup); err == nil {
		t.Fatal("match items must be distinct")
	}
}

func TestOldCheckpointsStillDecode(t *testing.T) {
	raw := `[{"id":"q1","type":"mcq","objectiveId":"o","prompt":"p","options":["a","b"],"answer":1,"tokens":null,"solution":"","explanation":"-","acceptedAnswers":["b"],"rubric":""}]`
	var items []CheckItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatal(err)
	}
	if s, ok := FixedVerdict(items[0], "1", nil); !ok || s != 100 || items[0].Key() != "b" {
		t.Fatalf("an old item grades as before: %+v", items[0])
	}
}
