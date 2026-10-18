package tutor

import (
	"os"
	"strings"
	"testing"

	"fluentpath/api/internal/curriculum"
)

func TestRecognitionVocabularyRejectsUntaughtAnswersAndDistractors(t *testing.T) {
	bank := []string{"chip", "cheap", "seat"}
	for _, typ := range []string{QuestionMCQ, QuestionMulti} {
		for _, options := range [][]string{{"leave", "live", "life"}, {"chip", "cheap", "leave"}} {
			q := CheckItem{ID: "q1", Question: Question{Type: typ, Options: options}}
			if err := ValidateRecognitionVocabulary([]CheckItem{q}, bank); err == nil || !strings.Contains(err.Error(), "taught vocabulary") {
				t.Fatalf("%s accepted untaught recognition choices %v: %v", typ, options, err)
			}
		}
	}
	for _, options := range [][]string{{"Chip", "cheap", "seat"}, {"Put your tongue between your teeth", "Keep your tongue behind your teeth"}} {
		q := CheckItem{Question: Question{Type: QuestionMCQ, Options: options}}
		if err := ValidateRecognitionVocabulary([]CheckItem{q}, bank); err != nil {
			t.Fatalf("valid taught words or explanatory choices rejected: %v", err)
		}
	}
	if err := ValidateRecognitionVocabulary([]CheckItem{{Question: Question{Type: QuestionMCQ, Options: []string{"other"}}}}, nil); err != nil {
		t.Fatal("unconfigured lessons must retain their own scope rules", err)
	}
}

func TestRecognitionExposureCannotComeFromPracticeOrFlashcards(t *testing.T) {
	p := Plan{Steps: []Step{
		{Kind: StepQuestion, Text: "leave", Question: &Question{Options: []string{"leave"}}},
		{Kind: StepFlashcards, Flashcards: []Card{{Front: "leave", Back: "pergi"}}},
	}}
	if err := validateRecognitionExposure(p, []string{"leave"}); err == nil {
		t.Fatal("a quiz or flashcard alone is not prior teaching")
	}
	p.Steps = append(p.Steps, Step{Kind: StepConcept, Pairs: []Pair{{EN: "leave", ID: "pergi · ii panjang"}}})
	if err := validateRecognitionExposure(p, []string{"leave"}); err != nil {
		t.Fatal(err)
	}
}

func TestSoundsCheckpointUsesTaughtWordsAndExistingAudio(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	l, _ := c.Lesson("a0-sounds-indonesian")
	if len(l.RecognitionVocabulary) == 0 {
		t.Fatal("phonics needs an explicit taught recognition bank")
	}
	for _, lang := range []string{"en", "id"} {
		p, ok, err := AuthoredPlan(c, l.NodeID, lang)
		if err != nil || !ok {
			t.Fatalf("plan: %v", err)
		}
		if err := validateRecognitionExposure(p, l.RecognitionVocabulary); err != nil {
			t.Fatal(lang, err)
		}
		items, err := AuthoredItems(l, lang)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateRecognitionVocabulary(items, l.RecognitionVocabulary); err != nil {
			t.Fatal(lang, err)
		}
		for _, item := range items {
			if item.Listen != "" {
				if _, err := os.Stat(c.AudioPath(item.Listen)); err != nil {
					t.Fatalf("%s needs recorded audio; no live TTS: %v", item.Listen, err)
				}
			}
		}
	}
}
