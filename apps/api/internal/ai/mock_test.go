package ai

import (
	"strings"
	"testing"

	"fluentpath/api/internal/curriculum"
)

func TestMatches(t *testing.T) {
	cases := []struct {
		answer   string
		expected []string
		want     bool
	}{
		{"She is.", []string{"is"}, true},
		{"C-A-T", []string{"c a t"}, true},
		{"an", []string{"a"}, false},
		{"They aren’t at school!", []string{"they aren't at school"}, true},
		{"", []string{"x"}, false},
		{"I get up at 6 every day", []string{"i get up at"}, true},
	}
	for _, c := range cases {
		if got := Matches(c.answer, c.expected); got != c.want {
			t.Errorf("Matches(%q, %v) = %v, want %v", c.answer, c.expected, got, c.want)
		}
	}
}

func TestTutorPromptIncludesLanguagePolicy(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	n, _ := c.Node("a0-pronouns")
	l, _ := c.Lesson("a0-pronouns")
	p := tutorSystemPrompt(TutorInput{Policy: c.Tutor.Policy, LanguagePolicy: c.Tutor.LanguagePolicy["id"], Locale: "id", Node: n, Lesson: l, Phase: "INTRODUCTION", NextPhase: "EXPLANATION"})
	if !strings.Contains(p, "## Language") || !strings.Contains(p, "mainly in Indonesian") {
		t.Fatal("ID language policy missing from tutor prompt")
	}
	e := evalSystemPrompt(EvalInput{Policy: c.Tutor.EvaluatorPolicy, LanguagePolicy: c.Tutor.LanguagePolicy["id"]})
	if !strings.Contains(e, "mainly in Indonesian") {
		t.Fatal("ID language policy missing from evaluator prompt")
	}
}
