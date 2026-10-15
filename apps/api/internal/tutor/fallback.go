package tutor

import (
	"strings"

	"fluentpath/api/internal/curriculum"
)

var fallbackText = map[string]map[string]string{
	"en": {"question": "Your turn", "try": "Try this one.", "examples": "More examples", "examplesText": "Listen and read these out loud.",
		"cards": "Flashcards", "cardsText": "Save these to review later.", "sheet": "Cheat sheet", "sheetText": "Here's everything on one page.",
		"resources": "See it in real life", "resourcesText": "Real-world videos and pages for this lesson.", "topic": "Topic", "meaning": "Meaning"},
	"id": {"question": "Giliranmu", "try": "Coba yang ini.", "examples": "Contoh lain", "examplesText": "Dengarkan dan baca dengan suara keras.",
		"cards": "Flashcard", "cardsText": "Simpan untuk diulang nanti.", "sheet": "Cheat sheet", "sheetText": "Semuanya dalam satu halaman.",
		"resources": "Lihat di dunia nyata", "resourcesText": "Video dan halaman nyata untuk pelajaran ini.", "topic": "Topik", "meaning": "Arti"},
}

// fit shortens reading material (which may be long) to a step's limit,
// cutting after the last whole paragraph or sentence that fits.
func fit(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	head := string(r[:max-1])
	for _, sep := range []string{"\n", ". "} {
		if i := strings.LastIndex(head, sep); i > len(head)/2 {
			return strings.TrimSpace(head[:i+len(sep)])
		}
	}
	return strings.TrimSpace(head) + "…"
}

func ft(lang, key string) string {
	if m, ok := fallbackText[lang]; ok {
		return m[key]
	}
	return fallbackText["en"][key]
}

// QuestionFromPrompt turns a practice prompt into a multiple-choice question,
// using other prompts' answers as distractors (or a word-order question when
// there aren't enough distractors). ok is false if neither works.
func QuestionFromPrompt(p curriculum.PracticePrompt, all []curriculum.PracticePrompt, rotate int) (Question, bool) {
	if len(p.Expected) == 0 {
		return Question{}, false
	}
	answer := p.Expected[0]
	opts := []string{answer}
	for j := 0; j < len(all) && len(opts) < 3; j++ {
		o := all[(j+rotate)%len(all)]
		if len(o.Expected) == 0 {
			continue
		}
		d := o.Expected[0]
		dup := false
		for _, x := range opts {
			if NormalizeWords(x) == NormalizeWords(d) {
				dup = true
			}
		}
		if !dup {
			opts = append(opts, d)
		}
	}
	explain := p.Hint
	if explain == "" {
		explain = answer
	}
	if len(opts) >= 2 {
		k := rotate % len(opts)
		opts = append(opts[k:], opts[:k]...)
		return Question{Type: QuestionMCQ, ObjectiveID: p.ObjectiveID, Prompt: p.Prompt, Options: opts, Answer: (len(opts) - k) % len(opts), Explanation: explain}, true
	}
	words := strings.Fields(answer)
	if len(words) < 2 || len(words) > maxTokens {
		return Question{}, false
	}
	tokens := append([]string{}, words...)
	for i, j := 0, len(tokens)-1; i < j; i, j = i+1, j-1 {
		tokens[i], tokens[j] = tokens[j], tokens[i]
	}
	return Question{Type: QuestionWordOrder, ObjectiveID: p.ObjectiveID, Prompt: p.Prompt, Tokens: tokens, Solution: answer, Explanation: explain}, true
}

// FallbackPlan builds a plan straight from the lesson's material, used by the
// offline provider and whenever the AI's plan can't be used.
func FallbackPlan(node *curriculum.Node, lesson *curriculum.Lesson, mat *curriculum.MaterialView, lang string) Plan {
	p := Plan{Greeting: mat.Lead}
	prompts := append(append([]curriculum.PracticePrompt{}, lesson.GuidedPractice...), lesson.IndependentPrompt...)
	questionFor := func(obj string, rotate int) *Step {
		for _, pr := range prompts {
			if pr.ObjectiveID != obj {
				continue
			}
			if q, ok := QuestionFromPrompt(pr, prompts, rotate); ok {
				return &Step{Kind: StepQuestion, Title: ft(lang, "question"), Text: ft(lang, "try"), Question: &q}
			}
		}
		return nil
	}
	concept := func(s curriculum.Section) Step {
		st := Step{Kind: StepConcept, Title: s.Title}
		for _, b := range s.Blocks {
			switch b.Type {
			case curriculum.BlockText:
				if st.Body == "" {
					st.Body = fit(b.Text, maxText)
				}
			case curriculum.BlockTip:
				if st.Text == "" {
					st.Text = fit(b.Text, maxText)
				}
			case curriculum.BlockExamples:
				for _, r := range b.Rows {
					if len(st.Pairs) < maxPairs {
						st.Pairs = append(st.Pairs, Pair{EN: r.EN, ID: r.ID})
					}
				}
			}
		}
		if st.Body == "" {
			st.Body = s.Title
		}
		return st
	}
	objs := node.Objectives
	sections := mat.Sections
	for i, s := range sections {
		p.Steps = append(p.Steps, concept(s))
		if i < len(objs) {
			if q := questionFor(objs[i].ID, i); q != nil {
				p.Steps = append(p.Steps, *q)
			}
		}
	}
	// Any objective still without a question gets one now.
	for i := len(sections); i < len(objs); i++ {
		if q := questionFor(objs[i].ID, i); q != nil {
			p.Steps = append(p.Steps, *q)
		}
	}
	var ex []Example
	for _, k := range lesson.KeyPoints {
		for _, e := range k.Examples {
			if len(ex) < 6 {
				ex = append(ex, Example{EN: e.EN, ID: e.ID})
			}
		}
	}
	if len(ex) > 0 && len(p.Steps) < MaxSteps {
		p.Steps = append(p.Steps, Step{Kind: StepExamples, Title: ft(lang, "examples"), Text: ft(lang, "examplesText"), Examples: ex})
	}
	if len(mat.Flashcards) > 0 && len(p.Steps) < MaxSteps {
		var cards []Card
		for _, c := range mat.Flashcards {
			if len(cards) < maxCards {
				cards = append(cards, Card{Front: c.Front, Back: c.Back})
			}
		}
		p.Steps = append(p.Steps, Step{Kind: StepFlashcards, Title: ft(lang, "cards"), Text: ft(lang, "cardsText"), Flashcards: cards})
	}
	if len(lesson.KeyPoints) > 0 && len(p.Steps) < MaxSteps {
		a := &Artifact{Title: ft(lang, "sheet") + " · " + node.Title, Headers: []string{ft(lang, "topic"), ft(lang, "meaning")}}
		for _, k := range lesson.KeyPoints {
			if len(a.Rows) < maxRows {
				a.Rows = append(a.Rows, []string{k.Title, k.Explanation})
			}
		}
		a.Examples = ex
		p.Steps = append(p.Steps, Step{Kind: StepArtifact, Title: ft(lang, "sheet"), Text: ft(lang, "sheetText"), Artifact: a})
	}
	if len(mat.Resources) > 0 && len(p.Steps) < MaxSteps {
		p.Steps = append(p.Steps, Step{Kind: StepResources, Title: ft(lang, "resources"), Text: ft(lang, "resourcesText")})
	}
	for len(p.Steps) > MaxSteps {
		p.Steps = p.Steps[:MaxSteps]
	}
	return p
}
