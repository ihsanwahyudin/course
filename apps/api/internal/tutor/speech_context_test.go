package tutor

import (
	"os"
	"slices"
	"testing"

	"fluentpath/api/internal/curriculum"
)

func TestAlphabetAndPronounHaveSeparateSpokenText(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"en", "id"} {
		for _, node := range []string{"a0-pronouns", "a0-basic-sentences", "a0-writing-basics", "a1-writing-about-me", "a0-alphabet-sounds", "a0-listening-first"} {
			p, _, err := AuthoredPlan(c, node, lang)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, step := range p.Steps {
				for _, pair := range step.Pairs {
					if pair.EN != "I" {
						continue
					}
					found = true
					want := "I"
					if node == "a0-alphabet-sounds" || node == "a0-listening-first" {
						want = "I, as in ink."
					}
					if pair.SpokenText() != want || !slices.Contains(SpeechTexts(p, 100), want) {
						t.Fatalf("%s/%s I uses the wrong audio: %q", node, lang, pair.SpokenText())
					}
					if _, err := os.Stat(c.AudioPath(want)); err != nil {
						t.Fatal("missing existing recording", err)
					}
				}
			}
			if !found {
				t.Fatal("expected an I concept tile", node)
			}
		}
	}
}
