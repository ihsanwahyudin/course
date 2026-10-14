package curriculum

import (
	"strings"
	"testing"
)

func TestValidatorChecksMaterial(t *testing.T) {
	bad := 3
	l := readyLesson("a", "o")
	l.Material = append(l.Material[:2], Section{ID: "one", Title: "Dup", Blocks: []Block{
		{Type: BlockCheck, Prompt: "q", Options: []string{"x", "y"}, Answer: &bad},
		{Type: "poem", Text: "roses"},
		{Type: BlockExamples},
	}})
	l.Resources = []Resource{
		{Kind: ResourceVideo, Title: "v", Source: "s", URL: "https://vimeo.com/123"},
		{Kind: ResourceArticle, Title: "a", Source: "s", URL: "http://example.com"},
		{Kind: "podcast", Title: "p", Source: "s", URL: "https://example.com"},
	}
	l.Flashcards = []Flashcard{{Front: "", Back: "x"}, {Front: strings.Repeat("x", FlashcardMax+1), Back: "y"}}
	err := build([]Node{{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentReady}}, map[string]*Lesson{"a": l})
	for _, want := range []string{
		`duplicate section "one"`,
		`check needs a prompt, 2–4 options and an answer index in range`,
		`unknown block type "poem"`,
		`examples need rows`,
		`video "https://vimeo.com/123" must be a YouTube link`,
		`resource "http://example.com" must be an https URL`,
		`unknown resource kind "podcast"`,
		`flashcards need a front and back`,
	} {
		wantErr(t, err, want)
	}

	l = readyLesson("a", "o")
	l.Material = l.Material[:2]
	err = build([]Node{{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentReady}}, map[string]*Lesson{"a": l})
	wantErr(t, err, `needs at least 3 material sections`)
}

func TestYouTubeID(t *testing.T) {
	for raw, want := range map[string]string{
		"https://www.youtube.com/watch?v=wlbQLBhLk_4":   "wlbQLBhLk_4",
		"https://youtu.be/wlbQLBhLk_4":                  "wlbQLBhLk_4",
		"https://m.youtube.com/watch?v=wlbQLBhLk_4&t=3": "wlbQLBhLk_4",
		"https://www.youtube.com/shorts/BMdneLySfnE":    "BMdneLySfnE",
		"http://www.youtube.com/watch?v=wlbQLBhLk_4":    "",
		"https://evil.com/watch?v=wlbQLBhLk_4":          "",
		"https://www.youtube.com/watch?v=short":         "",
	} {
		id, ok := YouTubeID(raw)
		if (want == "") == ok || (ok && id != want) {
			t.Errorf("%s: got %q %v, want %q", raw, id, ok, want)
		}
	}
}

func TestRepositoryMaterial(t *testing.T) {
	c, err := LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range c.Nodes {
		if !n.Ready() {
			if c.MaterialIn(n.ID, "en") != nil {
				t.Fatalf("%s has no lesson, so no material", n.ID)
			}
			continue
		}
		en, id := c.MaterialIn(n.ID, "en"), c.MaterialIn(n.ID, "id")
		if len(en.Sections) < 3 || len(en.Resources) == 0 || len(en.Flashcards) == 0 || len(en.SuggestedQuestions) == 0 {
			t.Fatalf("%s: material incomplete: %+v", n.ID, en)
		}
		if en.Sections[0].Blocks[0].Text == id.Sections[0].Blocks[0].Text || en.SuggestedQuestions[0] == id.SuggestedQuestions[0] || en.Lead == id.Lead {
			t.Fatalf("%s: Indonesian material not applied", n.ID)
		}
		if id.Sections[0].I18n != nil || id.Sections[0].Blocks[0].I18n != nil || id.Resources[0].I18n != nil {
			t.Fatalf("%s: translation maps leak into the view", n.ID)
		}
	}
	l, _ := c.Lesson("a1-to-be")
	if txt := l.MaterialText(); !strings.Contains(txt, "### am, is, are") || !strings.Contains(txt, "I am a student.") {
		t.Fatalf("material text for the prompt: %s", txt)
	}
}
