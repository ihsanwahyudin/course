package tutor

import "testing"

func TestLevelQuestionInteractions(t *testing.T) {
	q := CheckItem{ID: "spell", Question: Question{Type: QuestionLetters, Solution: "cat", Tokens: []string{"A", "C", "T", "G"}}}
	if err := ValidateLevelQuestions([]CheckItem{q}, "A0"); err == nil {
		t.Fatal("A0 must not have decoy letters")
	}
	if err := ValidateLevelQuestions([]CheckItem{q}, "A1"); err != nil {
		t.Fatal(err)
	}
	q.Tokens = append(q.Tokens, "J", "E")
	if err := ValidateLevelQuestions([]CheckItem{q}, "A1"); err == nil {
		t.Fatal("A1 allows at most two decoys")
	}
	if err := ValidateLevelQuestions([]CheckItem{q}, "B1"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"Spell green with dashes between its letters.", "Write the words that need a capital letter.", "Tulis huruf-huruf vokal dalam kata table."} {
		q.Question = Question{Type: QuestionText, Prompt: p}
		if err := ValidateLevelQuestions([]CheckItem{q}, "A0"); err == nil {
			t.Errorf("formatted list or spelling must use a picker: %s", p)
		}
	}
	q.Prompt = "Rewrite the date in the house style: day, month, year."
	if err := ValidateLevelQuestions([]CheckItem{q}, "C2"); err != nil {
		t.Fatal("a genuine format objective must remain possible", err)
	}
}
