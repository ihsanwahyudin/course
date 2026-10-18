package curriculum

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Block types inside a material section.
const (
	BlockText     = "text"     // paragraph; **bold** and `code` allowed
	BlockExamples = "examples" // English sentences with an Indonesian gloss
	BlockTip      = "tip"
	BlockCheck    = "check" // one multiple-choice practice question (not graded)
)

// Resource kinds.
const (
	ResourceVideo   = "video"
	ResourceArticle = "article"
	ResourceImage   = "image"
)

// Section is one part of a lesson's reading material.
type Section struct {
	ID     string                 `json:"id"`
	Title  string                 `json:"title"`
	Blocks []Block                `json:"blocks"`
	I18n   map[string]SectionText `json:"i18n,omitempty"`
}

type SectionText struct {
	Title string `json:"title"`
}

type ExampleRow struct {
	EN   string `json:"en"`
	ID   string `json:"id"` // Indonesian gloss
	Note string `json:"note,omitempty"`
	// I18n translates the note.
	I18n map[string]RowI18n `json:"i18n,omitempty"`
}

type RowI18n struct {
	Note string `json:"note,omitempty"`
}

type Block struct {
	Type string `json:"type"`
	// text / tip
	Text string `json:"text,omitempty"`
	// examples
	Rows []ExampleRow `json:"rows,omitempty"`
	// check: English being tested stays English. Explanatory choices may
	// have translated labels in I18n, preserving their indexes.
	Prompt  string               `json:"prompt,omitempty"`
	Options []string             `json:"options,omitempty"`
	Answer  *int                 `json:"answer,omitempty"`
	Explain string               `json:"explain,omitempty"`
	I18n    map[string]BlockI18n `json:"i18n,omitempty"`
}

type BlockI18n struct {
	Text    string   `json:"text,omitempty"`
	Prompt  string   `json:"prompt,omitempty"`
	Explain string   `json:"explain,omitempty"`
	Options []string `json:"options,omitempty"`
}

// Resource is a hand-picked real-world link shown with the lesson.
type Resource struct {
	Kind   string                  `json:"kind"`
	Title  string                  `json:"title"`
	URL    string                  `json:"url"`
	Source string                  `json:"source"`
	Note   string                  `json:"note,omitempty"`
	I18n   map[string]ResourceText `json:"i18n,omitempty"`
}

type ResourceText struct {
	Note string `json:"note,omitempty"`
}

// Flashcard is a front/back pair; lessons ship a starter deck.
type Flashcard struct {
	Front string `json:"front"`
	Back  string `json:"back"`
}

// LessonText holds a lesson's translated text.
type LessonText struct {
	Intro              string   `json:"intro"`
	SuggestedQuestions []string `json:"suggestedQuestions,omitempty"`
	// Assessment maps a checkpoint item's id to its prompt in this language.
	// Canonical answers and rubrics stay in English for grading.
	Assessment map[string]string `json:"assessment,omitempty"`
	// AssessmentOptions translates explanatory choices, preserving their indexes.
	AssessmentOptions map[string][]string `json:"assessmentOptions,omitempty"`
}

// MaterialView is a lesson's reading material in one language.
type MaterialView struct {
	Lead               string      `json:"lead"`
	Sections           []Section   `json:"sections"`
	SuggestedQuestions []string    `json:"suggestedQuestions"`
	Resources          []Resource  `json:"resources"`
	Flashcards         []Flashcard `json:"flashcards"`
}

// MaterialIn returns the lesson's material with text in lang (English
// fallback). Translation maps are stripped from the copy.
func (c *Curriculum) MaterialIn(nodeID, lang string) *MaterialView {
	l, ok := c.lessons[nodeID]
	if !ok {
		return nil
	}
	lang = c.Lang(lang)
	v := &MaterialView{Lead: l.IntroIn(lang), Sections: []Section{}, SuggestedQuestions: l.SuggestedQuestions,
		Resources: []Resource{}, Flashcards: l.Flashcards}
	if t, ok := l.I18n[lang]; ok && len(t.SuggestedQuestions) > 0 {
		v.SuggestedQuestions = t.SuggestedQuestions
	}
	if v.SuggestedQuestions == nil {
		v.SuggestedQuestions = []string{}
	}
	if v.Flashcards == nil {
		v.Flashcards = []Flashcard{}
	}
	for _, s := range l.Material {
		ls := Section{ID: s.ID, Title: s.Title}
		if t, ok := s.I18n[lang]; ok && t.Title != "" {
			ls.Title = t.Title
		}
		for _, b := range s.Blocks {
			lb := b
			lb.I18n = nil
			if t, ok := b.I18n[lang]; ok {
				if t.Text != "" {
					lb.Text = t.Text
				}
				if t.Prompt != "" {
					lb.Prompt = t.Prompt
				}
				if t.Explain != "" {
					lb.Explain = t.Explain
				}
				if len(t.Options) > 0 {
					lb.Options = append([]string{}, t.Options...)
				}
			}
			if len(b.Rows) > 0 {
				lb.Rows = make([]ExampleRow, len(b.Rows))
				for i, r := range b.Rows {
					if t, ok := r.I18n[lang]; ok && t.Note != "" {
						r.Note = t.Note
					}
					r.I18n = nil
					lb.Rows[i] = r
				}
			}
			ls.Blocks = append(ls.Blocks, lb)
		}
		v.Sections = append(v.Sections, ls)
	}
	for _, r := range l.Resources {
		lr := r
		lr.I18n = nil
		if t, ok := r.I18n[lang]; ok && t.Note != "" {
			lr.Note = t.Note
		}
		v.Resources = append(v.Resources, lr)
	}
	return v
}

// MaterialText flattens the English material for the tutor prompt.
func (l *Lesson) MaterialText() string {
	var b strings.Builder
	for _, s := range l.Material {
		fmt.Fprintf(&b, "### %s\n", s.Title)
		for _, bl := range s.Blocks {
			switch bl.Type {
			case BlockText, BlockTip:
				b.WriteString(bl.Text + "\n")
			case BlockExamples:
				for _, r := range bl.Rows {
					fmt.Fprintf(&b, "- %s (%s)\n", r.EN, r.ID)
				}
			case BlockCheck:
				fmt.Fprintf(&b, "Practice: %s [%s]\n", bl.Prompt, strings.Join(bl.Options, " / "))
			}
		}
	}
	return b.String()
}

// RecognitionVocabulary must be supported by teaching, not merely appear in
// a quiz or teacher notes. These checks are lexical evidence, not a substitute
// for reviewing whether the pronunciation is actually explained.
var recognitionVocabularyWord = regexp.MustCompile(`^[a-z]+(?:'[a-z]+)?$`)

func validateRecognitionVocabulary(l *Lesson) []error {
	var errs []error
	var teaching strings.Builder
	for _, section := range l.Material {
		for _, block := range section.Blocks {
			switch block.Type {
			case BlockText, BlockTip:
				teaching.WriteString(block.Text + "\n")
			case BlockExamples:
				for _, row := range block.Rows {
					teaching.WriteString(row.EN + "\n")
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, word := range l.RecognitionVocabulary {
		if !recognitionVocabularyWord.MatchString(word) || seen[word] {
			errs = append(errs, fmt.Errorf("lesson %q: recognition vocabulary needs distinct lowercase English words, got %q", l.NodeID, word))
			continue
		}
		seen[word] = true
		if !regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`).MatchString(teaching.String()) {
			errs = append(errs, fmt.Errorf("lesson %q: recognition word %q is missing from teaching material", l.NodeID, word))
		}
	}
	return errs
}

var youTubeIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// YouTubeID extracts the video id from a youtube.com/watch or youtu.be URL.
func YouTubeID(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return "", false
	}
	var id string
	switch strings.TrimPrefix(u.Hostname(), "www.") {
	case "youtube.com", "m.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		} else if strings.HasPrefix(u.Path, "/shorts/") || strings.HasPrefix(u.Path, "/embed/") {
			id = u.Path[strings.LastIndex(u.Path, "/")+1:]
		}
	case "youtu.be":
		id = strings.TrimPrefix(u.Path, "/")
	}
	return id, youTubeIDRe.MatchString(id)
}

// HTTPSURL reports whether raw is an absolute https URL with a host.
func HTTPSURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}

func validateMaterial(c *Curriculum) []error {
	var errs []error
	for id, l := range c.lessons {
		if len(l.Material) < 3 {
			errs = append(errs, fmt.Errorf("lesson %q: needs at least 3 material sections", id))
		}
		seen := map[string]bool{}
		for _, s := range l.Material {
			if s.ID == "" || strings.TrimSpace(s.Title) == "" || len(s.Blocks) == 0 {
				errs = append(errs, fmt.Errorf("lesson %q: section needs id, title and blocks", id))
			}
			if seen[s.ID] {
				errs = append(errs, fmt.Errorf("lesson %q: duplicate section %q", id, s.ID))
			}
			seen[s.ID] = true
			for i, b := range s.Blocks {
				where := fmt.Sprintf("lesson %q section %q block %d", id, s.ID, i)
				for lang, text := range b.I18n {
					if len(text.Options) == 0 {
						continue
					}
					if b.Type != BlockCheck {
						errs = append(errs, fmt.Errorf("%s: %q option labels require a check block", where, lang))
					} else if err := validateOptionLabels(b.Options, text.Options); err != nil {
						errs = append(errs, fmt.Errorf("%s: %q %w", where, lang, err))
					}
				}
				switch b.Type {
				case BlockText, BlockTip:
					if strings.TrimSpace(b.Text) == "" {
						errs = append(errs, fmt.Errorf("%s: empty %s", where, b.Type))
					}
				case BlockExamples:
					if len(b.Rows) == 0 {
						errs = append(errs, fmt.Errorf("%s: examples need rows", where))
					}
					for _, r := range b.Rows {
						if strings.TrimSpace(r.EN) == "" || strings.TrimSpace(r.ID) == "" {
							errs = append(errs, fmt.Errorf("%s: example rows need en and id", where))
						}
					}
				case BlockCheck:
					if strings.TrimSpace(b.Prompt) == "" || len(b.Options) < 2 || len(b.Options) > 4 || b.Answer == nil || *b.Answer < 0 || *b.Answer >= len(b.Options) {
						errs = append(errs, fmt.Errorf("%s: check needs a prompt, 2–4 options and an answer index in range", where))
					}
				default:
					errs = append(errs, fmt.Errorf("%s: unknown block type %q", where, b.Type))
				}
			}
		}
		for _, r := range l.Resources {
			if strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.Source) == "" {
				errs = append(errs, fmt.Errorf("lesson %q: resource needs title and source", id))
			}
			if !HTTPSURL(r.URL) {
				errs = append(errs, fmt.Errorf("lesson %q: resource %q must be an https URL", id, r.URL))
			}
			switch r.Kind {
			case ResourceVideo:
				if _, ok := YouTubeID(r.URL); !ok {
					errs = append(errs, fmt.Errorf("lesson %q: video %q must be a YouTube link", id, r.URL))
				}
			case ResourceArticle, ResourceImage:
			default:
				errs = append(errs, fmt.Errorf("lesson %q: unknown resource kind %q", id, r.Kind))
			}
		}
		for _, f := range l.Flashcards {
			if strings.TrimSpace(f.Front) == "" || strings.TrimSpace(f.Back) == "" || len([]rune(f.Front)) > FlashcardMax || len([]rune(f.Back)) > FlashcardMax {
				errs = append(errs, fmt.Errorf("lesson %q: flashcards need a front and back of at most %d characters", id, FlashcardMax))
			}
		}
		for _, lang := range c.Languages {
			if lang == BaseLanguage {
				continue
			}
			for _, s := range l.Material {
				if strings.TrimSpace(s.I18n[lang].Title) == "" {
					errs = append(errs, fmt.Errorf("lesson %q section %q: missing %q title", id, s.ID, lang))
				}
				for i, b := range s.Blocks {
					t := b.I18n[lang]
					if ((b.Type == BlockText || b.Type == BlockTip) && strings.TrimSpace(t.Text) == "") ||
						(b.Explain != "" && strings.TrimSpace(t.Explain) == "") {
						errs = append(errs, fmt.Errorf("lesson %q section %q block %d: missing %q text", id, s.ID, i, lang))
					}
				}
			}
			if len(l.SuggestedQuestions) > 0 && len(l.I18n[lang].SuggestedQuestions) != len(l.SuggestedQuestions) {
				errs = append(errs, fmt.Errorf("lesson %q: missing %q suggested questions", id, lang))
			}
			for _, r := range l.Resources {
				if r.Note != "" && strings.TrimSpace(r.I18n[lang].Note) == "" {
					errs = append(errs, fmt.Errorf("lesson %q: resource %q missing %q note", id, r.URL, lang))
				}
			}
		}
	}
	return errs
}

// FlashcardMax caps each side of a flashcard.
const FlashcardMax = 160

// Translations preserve the number and index of the canonical choices.
func validateOptionLabels(options, labels []string) error {
	if len(labels) != len(options) {
		return fmt.Errorf("translated option labels must match every option")
	}
	seen := map[string]bool{}
	for _, label := range labels {
		key := strings.ToLower(strings.TrimSpace(label))
		if key == "" || len([]rune(label)) > 200 || seen[key] {
			return fmt.Errorf("translated option labels must be distinct, non-empty and at most 200 characters")
		}
		seen[key] = true
	}
	return nil
}
