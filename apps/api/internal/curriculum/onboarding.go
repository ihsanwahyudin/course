package curriculum

import (
	"fmt"
	"sort"
	"strings"
)

// GoalQuestion is the onboarding question whose answer sets the learner's
// target level and matches node `focus` tags.
const GoalQuestion = "goal"

// FreeTextMax caps what a learner can type into a free-text option.
const FreeTextMax = 80

// Onboarding is the per-course questionnaire shown when a learner starts a
// course. Answers personalise the tutor and roadmap; they never change
// objectives, assessments or mastery rules.
type Onboarding struct {
	Version   int        `json:"version"`
	Questions []Question `json:"questions"`
}

type Question struct {
	ID       string                  `json:"id"`
	Title    string                  `json:"title"`
	Help     string                  `json:"help,omitempty"`
	Required bool                    `json:"required,omitempty"`
	Options  []Option                `json:"options"`
	I18n     map[string]QuestionText `json:"i18n,omitempty"`
}

type QuestionText struct {
	Title string `json:"title"`
	Help  string `json:"help,omitempty"`
}

type Option struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	// TargetLevel (goal options) becomes the enrollment's target level.
	TargetLevel string `json:"targetLevel,omitempty"`
	// FreeText lets the learner describe this option in their own words.
	FreeText bool `json:"freeText,omitempty"`
	// TutorHint is author-written guidance added to the tutor prompt.
	TutorHint string `json:"tutorHint,omitempty"`
	// IntroLine is appended to a lesson's opening message.
	IntroLine string                `json:"introLine,omitempty"`
	I18n      map[string]OptionText `json:"i18n,omitempty"`
}

type OptionText struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	IntroLine   string `json:"introLine,omitempty"`
}

// Profile is a learner's onboarding answers: question id -> option id, plus
// "<question id>_text" for free-text options.
type Profile map[string]string

// FreeTextKey is where a free-text answer to question qid is stored.
func FreeTextKey(qid string) string { return qid + "_text" }

func (c *Curriculum) question(qid string) *Question {
	if c.Onboarding == nil {
		return nil
	}
	for i := range c.Onboarding.Questions {
		if c.Onboarding.Questions[i].ID == qid {
			return &c.Onboarding.Questions[i]
		}
	}
	return nil
}

// Option returns the chosen option for a question in a profile, if any.
func (c *Curriculum) Option(p Profile, qid string) *Option {
	q := c.question(qid)
	if q == nil || p[qid] == "" {
		return nil
	}
	for i := range q.Options {
		if q.Options[i].ID == p[qid] {
			return &q.Options[i]
		}
	}
	return nil
}

// LocalQuestions returns the onboarding questions with text in lang. Tutor
// hints are left out: they are for the AI, not the learner.
func (c *Curriculum) LocalQuestions(lang string) []Question {
	if c.Onboarding == nil {
		return []Question{}
	}
	lang = c.Lang(lang)
	out := make([]Question, 0, len(c.Onboarding.Questions))
	for _, q := range c.Onboarding.Questions {
		lq := Question{ID: q.ID, Title: q.Title, Help: q.Help, Required: q.Required}
		if t, ok := q.I18n[lang]; ok {
			lq.Title, lq.Help = t.Title, t.Help
		}
		for _, o := range q.Options {
			lo := c.LocalOption(&o, lang)
			lo.TutorHint, lo.IntroLine = "", ""
			lq.Options = append(lq.Options, lo)
		}
		out = append(out, lq)
	}
	return out
}

// LocalOption returns a copy of an option with its text in lang.
func (c *Curriculum) LocalOption(o *Option, lang string) Option {
	lo := *o
	lo.I18n = nil
	if t, ok := o.I18n[c.Lang(lang)]; ok {
		lo.Label, lo.Description, lo.IntroLine = t.Label, t.Description, t.IntroLine
	}
	return lo
}

// TutorHints returns the author-written hints for a profile's answers, in
// question order, and the learner's own free-text descriptions.
func (c *Curriculum) TutorHints(p Profile) (hints, notes []string) {
	if c.Onboarding == nil {
		return nil, nil
	}
	for _, q := range c.Onboarding.Questions {
		o := c.Option(p, q.ID)
		if o == nil {
			continue
		}
		if o.TutorHint != "" {
			hints = append(hints, o.TutorHint)
		}
		if o.FreeText && strings.TrimSpace(p[FreeTextKey(q.ID)]) != "" {
			notes = append(notes, fmt.Sprintf("%s: %s", q.Title, p[FreeTextKey(q.ID)]))
		}
	}
	return hints, notes
}

// Recommended reports whether a node is tagged for the profile's goal.
func (c *Curriculum) Recommended(p Profile, n *Node) bool {
	goal := p[GoalQuestion]
	if goal == "" {
		return false
	}
	for _, f := range n.Focus {
		if f == goal {
			return true
		}
	}
	return false
}

// ValidateProfile checks answers against the questions and returns a
// cleaned copy (trimmed, free text only where allowed and capped).
func (c *Curriculum) ValidateProfile(answers map[string]string) (Profile, error) {
	if c.Onboarding == nil {
		return nil, fmt.Errorf("this course has no onboarding")
	}
	clean := Profile{}
	known := map[string]bool{}
	for _, q := range c.Onboarding.Questions {
		known[q.ID] = true
		known[FreeTextKey(q.ID)] = true
		v := strings.TrimSpace(answers[q.ID])
		if v == "" {
			if q.Required {
				return nil, fmt.Errorf("please answer %q", q.Title)
			}
			continue
		}
		var opt *Option
		for i := range q.Options {
			if q.Options[i].ID == v {
				opt = &q.Options[i]
			}
		}
		if opt == nil {
			return nil, fmt.Errorf("%q is not an option for %q", v, q.Title)
		}
		clean[q.ID] = v
		if text := cleanText(answers[FreeTextKey(q.ID)]); text != "" {
			if !opt.FreeText {
				return nil, fmt.Errorf("%q does not take a description", opt.Label)
			}
			if len([]rune(text)) > FreeTextMax {
				return nil, fmt.Errorf("keep the description under %d characters", FreeTextMax)
			}
			clean[FreeTextKey(q.ID)] = text
		}
	}
	for k := range answers {
		if !known[k] {
			return nil, fmt.Errorf("unknown question %q", k)
		}
	}
	return clean, nil
}

// cleanText trims and drops control characters (including newlines), so a
// free-text answer stays a single line of data inside the tutor prompt.
func cleanText(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func validateOnboarding(c *Curriculum) []error {
	var errs []error
	goals := map[string]bool{}
	if ob := c.Onboarding; ob != nil {
		if ob.Version < 1 {
			errs = append(errs, fmt.Errorf("onboarding: version must be at least 1"))
		}
		qids := map[string]bool{}
		for _, q := range ob.Questions {
			if q.ID == "" || strings.TrimSpace(q.Title) == "" {
				errs = append(errs, fmt.Errorf("onboarding: question needs id and title"))
			}
			if qids[q.ID] {
				errs = append(errs, fmt.Errorf("onboarding: duplicate question %q", q.ID))
			}
			qids[q.ID] = true
			if len(q.Options) < 2 {
				errs = append(errs, fmt.Errorf("onboarding: question %q needs at least two options", q.ID))
			}
			oids := map[string]bool{}
			for _, o := range q.Options {
				if o.ID == "" || strings.TrimSpace(o.Label) == "" {
					errs = append(errs, fmt.Errorf("onboarding: question %q has an option without id or label", q.ID))
				}
				if oids[o.ID] {
					errs = append(errs, fmt.Errorf("onboarding: question %q has duplicate option %q", q.ID, o.ID))
				}
				oids[o.ID] = true
				if o.TargetLevel != "" && c.LevelIndex(o.TargetLevel) < 0 {
					errs = append(errs, fmt.Errorf("onboarding: option %q targets unknown level %q", o.ID, o.TargetLevel))
				}
				if q.ID == GoalQuestion {
					goals[o.ID] = true
				}
			}
			for _, lang := range c.Languages {
				if lang == BaseLanguage {
					continue
				}
				if strings.TrimSpace(q.I18n[lang].Title) == "" || (q.Help != "" && strings.TrimSpace(q.I18n[lang].Help) == "") {
					errs = append(errs, fmt.Errorf("onboarding: question %q: missing %q title or help", q.ID, lang))
				}
				for _, o := range q.Options {
					t := o.I18n[lang]
					if strings.TrimSpace(t.Label) == "" ||
						(o.Description != "" && strings.TrimSpace(t.Description) == "") ||
						(o.IntroLine != "" && strings.TrimSpace(t.IntroLine) == "") {
						errs = append(errs, fmt.Errorf("onboarding: option %q.%q: missing %q text", q.ID, o.ID, lang))
					}
				}
			}
			if q.ID == GoalQuestion && !q.Required {
				errs = append(errs, fmt.Errorf("onboarding: the %q question must be required", GoalQuestion))
			}
		}
		if !qids[GoalQuestion] {
			errs = append(errs, fmt.Errorf("onboarding: needs a %q question", GoalQuestion))
		}
	}
	for _, n := range c.byID {
		for _, f := range n.Focus {
			if !goals[f] {
				errs = append(errs, fmt.Errorf("node %q: focus %q is not a goal option", n.ID, f))
			}
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}
