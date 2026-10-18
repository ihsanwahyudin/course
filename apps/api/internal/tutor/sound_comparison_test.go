package tutor

import (
	"strings"
	"testing"

	"fluentpath/api/internal/curriculum"
)

func TestSoundComparisonCannotUseItsReferenceAsTheAnswer(t *testing.T) {
	cases := []struct{ prompt, answer string }{
		{"Pilih jawaban yang benar. Kata mana yang punya bunyi pendek /ɪ/ seperti pada 'ship'?", "ship"},
		{"Kata lain mana yang punya bunyi seperti ship?", "Ship."},
		{"Kata mana yang punya bunyi /ɪ/ seperti di kata ‘ship’?", "ship"},
		{"Which word has the short /ɪ/ sound as in 'ship'?", "ship"},
		{"Which word has the same vowel sound as the word SHIP?", "ship"},
		{"Which word has a vowel like in *ship*?", "ship"},
		{"Which word rhymes with `cat`?", "cat"},
		{"Kata mana yang berima dengan \"cat\"?", "cat"},
	}
	for _, tc := range cases {
		t.Run(tc.prompt, func(t *testing.T) {
			q := Question{Type: QuestionMCQ, ObjectiveID: "o", Prompt: tc.prompt, Options: []string{tc.answer, "seat", "food"}, Answer: 0, Explanation: "An explanation."}
			if _, err := validateQuestion(q, map[string]bool{"o": true}, 400); err == nil || !strings.Contains(err.Error(), "must differ from reference word") {
				t.Fatalf("accepted circular MCQ: %v", err)
			}
			text := CheckItem{Question: Question{Type: QuestionText, ObjectiveID: "o", Prompt: tc.prompt}, Accepted: []string{"sit", tc.answer}, Rubric: "Only the taught short vowel."}
			if _, err := validateTextItem(text, map[string]bool{"o": true}); err == nil {
				t.Fatal("an accepted variant cannot reuse the reference either")
			}
		})
	}
	multi := CheckItem{Question: Question{Type: QuestionMulti, ObjectiveID: "o", Prompt: "Choose words with the same vowel sound as in 'ship'.", Options: []string{"sit", "ship", "seat"}}, Answers: []int{0, 1}}
	if _, err := validateTileItem(multi, map[string]bool{"o": true}); err == nil {
		t.Fatal("multi must not silently accept the reference word")
	}
}

func TestSoundComparisonDoesNotBanUsefulExamplesOrListedChoices(t *testing.T) {
	for _, tc := range []struct{ prompt, answer string }{
		{"Kata lain mana yang punya bunyi /ɪ/ seperti pada 'ship'?", "sit"},
		{"Which word has the long /uː/ sound, like in 'food': full or fool?", "fool"},
		{"Which word rhymes with 'cat'?", "hat"},
		{"Which letter's name sounds like the word 'eye'?", "I"},
		{"Which letter has the /eɪ/ sound? Write it as a capital letter.", "A"},
		{"For this vowel sound, I would like a short answer: which letter is first?", "A"},
		{"Which word has the same sound as ship?", "sit"},
		{"Which word has a short vowel: ship or sheep?", "ship"},
		{"Dengarkan. Kata mana yang kamu dengar?", "ship"},
		{"How many syllables are there in 'needed'?", "two"},
		{"Which sentence uses like correctly? I like ship models.", "I like ship models."},
	} {
		if err := validateSoundComparison(tc.prompt, []string{tc.answer}); err != nil {
			t.Errorf("valid question rejected: %s: %v", tc.prompt, err)
		}
	}
	q := Question{Type: QuestionMCQ, ObjectiveID: "o", Prompt: "Kata lain mana yang punya bunyi /ɪ/ seperti pada 'ship'?", Options: []string{"sit", "seat", "food"}, Answer: 0, Explanation: "Sit dan ship memakai /ɪ/."}
	if _, err := validateQuestion(q, map[string]bool{"o": true}, 400); err != nil {
		t.Fatal(err)
	}
}

func TestAuthoredMaterialAndPracticeHaveNoCircularSoundComparisons(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range c.Nodes {
		l, ok := c.Lesson(n.ID)
		if !ok {
			continue
		}
		practice := append(append([]curriculum.PracticePrompt{}, l.GuidedPractice...), l.IndependentPrompt...)
		for _, p := range practice {
			if err := validateSoundComparison(p.Prompt, p.Expected); err != nil {
				t.Errorf("%s practice: %v", n.ID, err)
			}
		}
		for _, lang := range []string{"en", "id"} {
			for _, section := range c.MaterialIn(n.ID, lang).Sections {
				for _, b := range section.Blocks {
					if b.Type != curriculum.BlockCheck {
						continue
					}
					if err := validateSoundComparison(b.Prompt, []string{b.Options[*b.Answer]}); err != nil {
						t.Errorf("%s/%s/%s: %v", n.ID, lang, section.ID, err)
					}
				}
			}
		}
	}
}
