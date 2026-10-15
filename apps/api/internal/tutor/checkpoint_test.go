package tutor

import (
	"encoding/json"
	"fmt"
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
