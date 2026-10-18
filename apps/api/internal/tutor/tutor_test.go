package tutor

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"fluentpath/api/internal/curriculum"
)

func validPlan() Plan {
	return Plan{Greeting: "Hi!", Steps: []Step{
		{Kind: StepConcept, Title: "Pronouns", Text: "Let's start.", Body: "he = laki-laki", Pairs: []Pair{{EN: "he", ID: "dia (lk)"}}},
		{Kind: StepExamples, Title: "Examples", Examples: []Example{{"He is busy.", "Dia sibuk."}}},
		{Kind: StepQuestion, Title: "Try", Question: &Question{Type: QuestionMCQ, ObjectiveID: "subject", Prompt: "Andi is my manager. ___ is busy.", Options: []string{"He", "She", "It"}, Answer: 0, Explanation: "Andi is a man."}},
		{Kind: StepQuestion, Title: "Order", Question: &Question{Type: QuestionWordOrder, ObjectiveID: "possessive", Prompt: "Tasnya (Rina) baru.", Tokens: []string{"new", "Her", "is", "bag"}, Solution: "Her bag is new.", Explanation: "her + noun"}},
	}}
}

func TestValidatePlan(t *testing.T) {
	p := validPlan()
	if err := ValidatePlan(&p, []string{"subject", "possessive"}); err != nil {
		t.Fatal(err)
	}
	// A plan that leaves an objective unpractised is rejected.
	p = validPlan()
	if err := ValidatePlan(&p, []string{"subject", "possessive", "reference"}); err == nil || !strings.Contains(err.Error(), `"reference" has no question`) {
		t.Fatalf("missing objective: %v", err)
	}
	bad := []func(*Plan){
		func(p *Plan) { p.Steps = p.Steps[:2] },
		func(p *Plan) { p.Steps[0].Kind = "song" },
		func(p *Plan) { p.Steps[2].Question.Answer = 5 },
		func(p *Plan) { p.Steps[2].Question.Options = []string{"He", "he"} },
		func(p *Plan) { p.Steps[3].Question.Solution = "Her bag is old." },
		func(p *Plan) { p.Steps[2].Question.ObjectiveID = "nope" },
		func(p *Plan) { p.Steps[1].Examples = nil },
	}
	for i, mutate := range bad {
		p := validPlan()
		mutate(&p)
		if err := ValidatePlan(&p, []string{"subject", "possessive"}); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
	raw, _ := json.Marshal(validPlan())
	if _, err := ParsePlan(raw, []string{"subject", "possessive"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePlan([]byte(`{`), nil); !errors.Is(err, ErrMalformed) {
		t.Fatal("bad JSON is malformed")
	}
}

func TestHintsMustNotLeakTheAnswer(t *testing.T) {
	q := *validPlan().Steps[2].Question
	if _, err := ParseHint([]byte(`{"hint":"Andi is a man, so use He."}`), q); !errors.Is(err, ErrMalformed) {
		t.Fatal("a hint naming the answer must be rejected")
	}
	if h, err := ParseHint([]byte(`{"hint":"Is Andi a man or a woman? Think about who is busy."}`), q); err != nil || h == "" {
		t.Fatalf("a fair hint: %q %v", h, err)
	}
	wo := *validPlan().Steps[3].Question
	if !LeaksAnswer("Try: her bag is new", wo) || LeaksAnswer("Start with the owner.", wo) {
		t.Fatal("word-order leak check wrong")
	}
}

func TestParseAnswerAndSummary(t *testing.T) {
	a, err := ParseAnswer([]byte(`{"answer":"did is past.","scope":"other_lesson","otherLesson":"Past Simple","remember":"has an interview on Friday"}`))
	if err != nil || a.Scope != ScopeOtherLesson || a.OtherLesson != "Past Simple" || a.Remember == "" {
		t.Fatalf("answer: %+v %v", a, err)
	}
	if a, _ = ParseAnswer([]byte(`{"answer":"x","scope":"weird","otherLesson":"y","remember":""}`)); a.Scope != ScopeIn {
		t.Fatal("unknown scope falls back to in")
	}
	if _, err := ParseAnswer([]byte(`{"answer":"  "}`)); !errors.Is(err, ErrMalformed) {
		t.Fatal("empty answer is malformed")
	}
	s, err := ParseSummary([]byte(`{"mastered":["a","",""],"review":["b"],"memory":["c","d","e","f"],"nextOpening":"Next!"}`))
	if err != nil || len(s.Mastered) != 1 || len(s.Memory) != 3 || s.NextOpening != "Next!" {
		t.Fatalf("summary: %+v %v", s, err)
	}
}

func TestNormalizeWords(t *testing.T) {
	if NormalizeWords(" They’re  HERE! ") != "they're here" {
		t.Fatal(NormalizeWords(" They’re  HERE! "))
	}
}

func TestFallbackPlanIsValidForEveryLesson(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range c.Nodes {
		l, ok := c.Lesson(n.ID)
		if !ok {
			continue
		}
		for _, lang := range []string{"en", "id"} {
			node := c.LocalNode(n.ID, lang)
			p := FallbackPlan(node, l, c.MaterialIn(n.ID, lang), lang)
			var objs []string
			for _, o := range node.Objectives {
				objs = append(objs, o.ID)
			}
			if err := ValidatePlan(&p, objs); err != nil {
				t.Errorf("%s/%s: %v", n.ID, lang, err)
			}
		}
	}
}

// Every ready lesson ships with hand-written steps. ONLY_NODE=<id> checks one.
func TestAuthoredPlans(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	if id := os.Getenv("ONLY_NODE"); id != "" {
		if err := ValidateLessonSteps(c, id); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := ValidateAuthored(c); err != nil {
		t.Fatal(err)
	}
}

// Lesson audio is committed to the repo, so deployments never make it again.
// After changing lessons: go run ./cmd/server bundle-audio, then commit.
func TestEveryLessonSentenceHasRecordedAudio(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	texts := LessonSpeechTexts(c)
	if len(texts) < 100 {
		t.Fatalf("expected the lessons' sentences, got %d", len(texts))
	}
	for _, s := range texts {
		if _, err := os.Stat(c.AudioPath(s)); err != nil {
			t.Errorf("no recorded audio for %q (run `go run ./cmd/server bundle-audio`)", s)
		}
	}
}

func TestFitCutsLongMaterialAtABoundary(t *testing.T) {
	long := strings.Repeat("One short sentence. ", 80)
	got := fit(long, maxText)
	if runes(got) > maxText || !strings.HasSuffix(got, ".") {
		t.Fatalf("fit: %d runes, ends %q", runes(got), got[len(got)-10:])
	}
	if fit("Short.", maxText) != "Short." {
		t.Fatal("short text should stay as it is")
	}
}

func TestHandWrittenItemsOfEveryType(t *testing.T) {
	l := &curriculum.Lesson{Assessment: []curriculum.AssessmentItem{
		{ID: "x-1", ObjectiveID: "o", Type: "letters", Prompt: "Listen and spell the word.", Listen: "green", Solution: "green", Rubric: "G-R-E-E-N"},
		{ID: "x-2", ObjectiveID: "o", Type: "multi", Prompt: "Tick the vowels in 'table'.", Options: []string{"T", "A", "B", "E"}, CorrectOptions: []string{"A", "E"}, Rubric: "A and E"},
		{ID: "x-3", ObjectiveID: "o", Type: "match", Prompt: "Match each colour.", Pairs: []curriculum.AssessmentPair{{Left: "red", Right: "merah"}, {Left: "blue", Right: "biru"}}, Rubric: "r"},
		{ID: "x-4", ObjectiveID: "o", Type: "mcq", Prompt: "Which letter sounds like 'eye'?", Options: []string{"A", "I", "E"}, CorrectOption: "I", Rubric: "I"},
		{ID: "x-5", ObjectiveID: "o", Prompt: "Say hello.", AcceptableAnswers: []string{"hello"}, Rubric: "hello"},
	}}
	items, err := AuthoredItems(l, "en")
	if err == nil {
		items, err = ValidateCheckpoint(items, map[string]bool{"o": true}, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"G-R-E-E-N", "A, E", "red → merah; blue → biru", "I", "hello"}
	for i, it := range items {
		if it.Key() != want[i] {
			t.Errorf("item %d (%s): key %q, want %q", i+1, it.Type, it.Key(), want[i])
		}
	}
	if items[4].Type != QuestionText {
		t.Error("an item without a type is open text")
	}
}
