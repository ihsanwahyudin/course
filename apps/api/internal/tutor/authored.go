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

// AuthoredDraft is a hand-written checkpoint item as a writer's draft, with
// its prompt in the learner's language.
func AuthoredDraft(it curriculum.AssessmentItem, prompt string) DraftItem {
	typ := it.Type
	if typ == "" {
		typ = QuestionText
	}
	d := DraftItem{
		CheckItem: CheckItem{
			Question: Question{Type: typ, ObjectiveID: it.ObjectiveID, Prompt: prompt, Listen: it.Listen, Explanation: it.Rubric,
				Options: slices.Clone(it.Options), Tokens: slices.Clone(it.Tokens), Solution: it.Solution},
			Accepted: it.AcceptableAnswers, Rubric: it.Rubric,
		},
		CorrectOption: it.CorrectOption, CorrectOptions: it.CorrectOptions,
	}
	for _, p := range it.Pairs {
		d.Pairs = append(d.Pairs, MatchPair(p))
	}
	return d
}

// AuthoredDrafts is a lesson's hand-written checkpoint as drafts.
func AuthoredDrafts(l *curriculum.Lesson, lang string) []DraftItem {
	ds := make([]DraftItem, 0, len(l.Assessment))
	for _, it := range l.Assessment {
		ds = append(ds, AuthoredDraft(it, l.PromptIn(it, lang)))
	}
	return ds
}

// AuthoredItems is a lesson's hand-written checkpoint, before validation.
func AuthoredItems(l *curriculum.Lesson, lang string) ([]CheckItem, error) {
	return FromDrafts(AuthoredDrafts(l, lang))
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
	objectives := map[string]bool{}
	for _, o := range n.Objectives {
		objectives[o.ID] = true
	}
	for _, lang := range courseLanguages(c) {
		// The hand-written checkpoint is the fallback when the AI can't write
		// one, so it has to pass the same checks.
		items, err := AuthoredItems(lesson, lang)
		if err == nil {
			_, err = ValidateCheckpoint(items, objectives, nil)
		}
		if err != nil {
			bad("%s/%s: hand-written checkpoint: %v", n.ID, lang, err)
		}
	}
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

// SameSteps reports whether two plans have the same steps and graded
// questions, so a learner's place and answers in one fit the other (only
// wording may differ: another language, or a corrected explanation).
func SameSteps(a, b Plan) bool {
	if len(a.Steps) != len(b.Steps) {
		return false
	}
	for i := range a.Steps {
		x, y := a.Steps[i], b.Steps[i]
		if x.Kind != y.Kind || (x.Question == nil) != (y.Question == nil) {
			return false
		}
		if x.Question != nil && !sameQuestion(*x.Question, *y.Question) {
			return false
		}
	}
	return true
}

// sameQuestion compares what's graded: type, objective, options, answer,
// tokens and solution (prompts and explanations may be translated).
func sameQuestion(a, b Question) bool {
	return a.Type == b.Type && a.ObjectiveID == b.ObjectiveID && a.Listen == b.Listen && slices.Equal(a.Options, b.Options) &&
		a.Answer == b.Answer && slices.Equal(a.Tokens, b.Tokens) && a.Solution == b.Solution
}

// SpeechTexts lists the English a plan reads aloud: word tiles, example
// sentences and cheat-sheet examples (deduplicated, at most limit).
func SpeechTexts(p Plan, limit int) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		t = curriculum.SpeechKey(t)
		if t == "" || seen[t] || runes(t) > MaxSpeechRunes || len(out) >= limit {
			return
		}
		seen[t] = true
		out = append(out, t)
	}
	for _, st := range p.Steps {
		if st.Question != nil {
			add(st.Question.Listen)
		}
		for _, x := range st.Pairs {
			add(x.EN)
		}
		for _, x := range st.Examples {
			add(x.EN)
		}
		if st.Artifact != nil {
			for _, x := range st.Artifact.Examples {
				add(x.EN)
			}
		}
	}
	return out
}

// MaxSpeechRunes is the longest sentence the speech endpoint reads.
const MaxSpeechRunes = 300

// LessonSpeechTexts lists every sentence the course's hand-written lessons
// read aloud (steps and checkpoint listening items), in all languages,
// deduplicated. Lessons whose audio is waiting for content review
// (audio/pending.json) are left out.
func LessonSpeechTexts(c *curriculum.Curriculum) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range c.Nodes {
		if c.AudioPending(n.ID) {
			continue
		}
		if l, ok := c.Lesson(n.ID); ok {
			for _, it := range l.Assessment {
				if t := curriculum.SpeechKey(it.Listen); t != "" && !seen[t] {
					seen[t] = true
					out = append(out, t)
				}
			}
		}
		for _, lang := range c.PlanLanguages(n.ID) {
			p, ok, err := AuthoredPlan(c, n.ID, lang)
			if !ok || err != nil {
				continue
			}
			for _, t := range SpeechTexts(p, 1000) {
				if !seen[t] {
					seen[t] = true
					out = append(out, t)
				}
			}
		}
	}
	return out
}
