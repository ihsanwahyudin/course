package tutor

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"fluentpath/api/internal/curriculum"
)

// Lessons ship with hand-written steps (packages/courses/<course>/plans),
// so they open instantly and every learner gets the same reviewed lesson.

func objectiveIDs(n *curriculum.Node) []string {
	ids := make([]string, len(n.Objectives))
	for i, o := range n.Objectives {
		ids[i] = o.ID
	}
	return ids
}

// AuthoredPlan returns a lesson's hand-written plan in a language. ok is
// false when the lesson has none in that language.
func AuthoredPlan(c *curriculum.Curriculum, nodeID, lang string) (p Plan, ok bool, err error) {
	raw, ok := c.AuthoredPlan(nodeID, lang)
	if !ok {
		return Plan{}, false, nil
	}
	node := c.LocalNode(nodeID, lang)
	if node == nil {
		return Plan{}, true, fmt.Errorf("no node %q", nodeID)
	}
	p, err = ParsePlan(raw, objectiveIDs(node))
	return p, true, err
}

func courseLanguages(c *curriculum.Curriculum) []string {
	langs := []string{curriculum.BaseLanguage}
	for _, l := range c.Languages {
		if !slices.Contains(langs, l) {
			langs = append(langs, l)
		}
	}
	return langs
}

// ValidateAuthored checks that every ready lesson has valid hand-written
// steps in every course language, that the languages match step for step
// (same kinds, questions and answers), that practice questions don't repeat
// checkpoint items, and that word-order tokens come shuffled.
func ValidateAuthored(c *curriculum.Curriculum) error {
	var errs []error
	for _, n := range c.Nodes {
		errs = append(errs, ValidateLessonSteps(c, n.ID))
	}
	return errors.Join(errs...)
}

// ValidateLessonSteps runs the ValidateAuthored checks for one lesson.
func ValidateLessonSteps(c *curriculum.Curriculum, nodeID string) error {
	var errs []error
	bad := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }
	n, ok := c.Node(nodeID)
	if !ok {
		return fmt.Errorf("no node %q", nodeID)
	}
	lesson, ok := c.Lesson(n.ID)
	if !ok || n.ContentStatus != curriculum.ContentReady {
		return nil
	}
	plans := map[string]Plan{}
	for _, lang := range courseLanguages(c) {
		p, ok, err := AuthoredPlan(c, n.ID, lang)
		switch {
		case !ok:
			bad("%s: no hand-written steps in %q", n.ID, lang)
			continue
		case err != nil:
			bad("%s/%s: %v", n.ID, lang, err)
			continue
		}
		mat := c.MaterialIn(n.ID, lang)
		for i, st := range p.Steps {
			if q := st.Question; q != nil {
				for _, it := range lesson.Assessment {
					if NormalizeWords(it.Prompt) == NormalizeWords(q.Prompt) {
						bad("%s/%s step %d: repeats checkpoint item %s", n.ID, lang, i+1, it.ID)
					}
				}
				if q.Type == QuestionWordOrder && NormalizeWords(strings.Join(q.Tokens, " ")) == NormalizeWords(q.Solution) {
					bad("%s/%s step %d: word-order tokens are already in order", n.ID, lang, i+1)
				}
			}
			if st.Kind == StepResources && (mat == nil || len(mat.Resources) == 0) {
				bad("%s/%s step %d: resources step but the lesson has no curated links", n.ID, lang, i+1)
			}
		}
		plans[lang] = p
	}
	base, ok := plans[curriculum.BaseLanguage]
	if !ok {
		return errors.Join(errs...)
	}
	for lang, p := range plans {
		if lang == curriculum.BaseLanguage {
			continue
		}
		if len(p.Steps) != len(base.Steps) {
			bad("%s: %q has %d steps, %q has %d", n.ID, lang, len(p.Steps), curriculum.BaseLanguage, len(base.Steps))
			continue
		}
		for i := range p.Steps {
			a, b := base.Steps[i], p.Steps[i]
			if a.Kind != b.Kind {
				bad("%s step %d: %s in %q but %s in %q", n.ID, i+1, a.Kind, curriculum.BaseLanguage, b.Kind, lang)
				continue
			}
			if a.Question != nil && !sameQuestion(*a.Question, *b.Question) {
				bad("%s step %d: the question differs between %q and %q", n.ID, i+1, curriculum.BaseLanguage, lang)
			}
		}
	}
	return errors.Join(errs...)
}

// sameQuestion compares what's graded: type, objective, options, answer,
// tokens and solution (prompts and explanations may be translated).
func sameQuestion(a, b Question) bool {
	return a.Type == b.Type && a.ObjectiveID == b.ObjectiveID && slices.Equal(a.Options, b.Options) &&
		a.Answer == b.Answer && slices.Equal(a.Tokens, b.Tokens) && a.Solution == b.Solution
}
