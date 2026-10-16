package tutor

import "testing"

func TestLegacyLetterListsGradeContent(t *testing.T) {
	q := CheckItem{Question: Question{Type: QuestionText, Prompt: "Tulis huruf-huruf vokal dalam kata table."}, Accepted: []string{"a e"}}
	for _, answer := range []string{"a dan e", "A, E", "E and A", "a e"} {
		if score, fixed := FixedVerdict(q, answer, func(string, []string) bool { return false }); !fixed || score != 100 {
			t.Errorf("%q should earn full credit: %d %v", answer, score, fixed)
		}
	}
	for _, answer := range []string{"a", "a dan e dan i", "a and the e", "a a e"} {
		if LetterListMatch(answer, q.Accepted) {
			t.Errorf("wrong list accepted: %s", answer)
		}
	}
	if LetterListMatch("tea dan coffee", []string{"tea and coffee"}) || LetterListMatch("e a t", []string{"eat"}) {
		t.Fatal("normal sentences and spelling must not be treated as letter lists")
	}
	q.Prompt, q.Accepted = "Spell house with letters separated by spaces.", []string{"h o u s e"}
	if _, fixed := FixedVerdict(q, "e s u o h", func(string, []string) bool { return false }); fixed {
		t.Fatal("legacy spelling must still assess letter order")
	}
}

func TestFormattingTargetIsNotNormalisedAway(t *testing.T) {
	q := CheckItem{Question: Question{Type: QuestionText, Prompt: "House style: write dates as day + month in words + year, with no comma."}, Accepted: []string{"3 october 2026"}}
	if _, fixed := FixedVerdict(q, "3, October, 2026", func(string, []string) bool { return true }); fixed {
		t.Fatal("normalised match must not bypass a formatting rubric")
	}
	q.Prompt = "Write the greeting (a comma is optional)."
	if score, fixed := FixedVerdict(q, "anything", func(string, []string) bool { return true }); !fixed || score != 100 {
		t.Fatal("optional formatting must remain content-graded")
	}
}

func TestOfflineFormatPreservesTargetCase(t *testing.T) {
	q := CheckItem{Question: Question{Type: QuestionText, Prompt: "Rewrite with correct capital letters."}, Accepted: []string{"My name is Budi."}}
	if !MatchFormattedText(q, "My name is Budi.") || MatchFormattedText(q, "my name is budi.") {
		t.Fatal("capitalization target must preserve case")
	}
}
