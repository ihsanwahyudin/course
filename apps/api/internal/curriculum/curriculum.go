// Package curriculum loads and validates the versioned roadmap JSON in
// packages/curriculum. The curriculum is the authoritative, read-only source of
// nodes, prerequisites, objectives and lesson content; nothing at runtime (and
// in particular no AI output) is allowed to change it.
package curriculum

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const (
	ContentReady      = "ready"
	ContentComingSoon = "coming_soon"

	DefaultMasteryThreshold = 80
)

type Objective struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Node struct {
	ID               string      `json:"id"`
	Level            string      `json:"level"`
	Title            string      `json:"title"`
	Summary          string      `json:"summary"`
	PrerequisiteIDs  []string    `json:"prerequisiteIds"`
	Objectives       []Objective `json:"objectives"`
	MasteryThreshold int         `json:"masteryThreshold"`
	ContentStatus    string      `json:"contentStatus"`
	// Focus lists the onboarding goals (goal option ids) this node matters
	// most for; untagged nodes are core for everyone.
	Focus []string `json:"focus,omitempty"`
	// Foundations are lessons of other courses this node builds on (for
	// test-prep tracks: the grammar and skills from the core roadmap).
	Foundations []NodeRef `json:"foundations,omitempty"`
	// Translations keyed by language; never sent to clients (views use
	// the localized copy from Curriculum.LocalNode).
	I18n map[string]NodeText `json:"i18n,omitempty"`
}

type NodeText struct {
	Title      string            `json:"title"`
	Summary    string            `json:"summary"`
	Objectives map[string]string `json:"objectives"` // objective id -> text
}

func (n *Node) Ready() bool { return n.ContentStatus == ContentReady }

type LevelInfo struct {
	Level       string `json:"level"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type LevelText struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type levelFile struct {
	Version     string               `json:"version"`
	Level       string               `json:"level"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	I18n        map[string]LevelText `json:"i18n,omitempty"`
	Nodes       []Node               `json:"nodes"`
}

type Example struct {
	EN string `json:"en"`
	ID string `json:"id"` // Indonesian gloss
}

type KeyPoint struct {
	Title       string    `json:"title"`
	Explanation string    `json:"explanation"`
	Examples    []Example `json:"examples"`
}

type PracticePrompt struct {
	ObjectiveID string   `json:"objectiveId"`
	Prompt      string   `json:"prompt"`
	Expected    []string `json:"expected"`
	Hint        string   `json:"hint"`
}

type AssessmentItem struct {
	ID                string   `json:"id"`
	ObjectiveID       string   `json:"objectiveId"`
	Prompt            string   `json:"prompt"`
	AcceptableAnswers []string `json:"acceptableAnswers"`
	Rubric            string   `json:"rubric"`
}

// Lesson is the teachable content for a ready node.
type Lesson struct {
	NodeID            string           `json:"nodeId"`
	Intro             string           `json:"intro"`
	TeachingNotes     []string         `json:"teachingNotes"`
	KeyPoints         []KeyPoint       `json:"keyPoints"`
	GuidedPractice    []PracticePrompt `json:"guidedPractice"`
	IndependentPrompt []PracticePrompt `json:"independentPractice"`
	Assessment        []AssessmentItem `json:"assessment"`
	// Reading material shown on the lesson page, with the AI helper beside it.
	Material           []Section             `json:"material"`
	SuggestedQuestions []string              `json:"suggestedQuestions,omitempty"`
	Resources          []Resource            `json:"resources,omitempty"`
	Flashcards         []Flashcard           `json:"flashcards,omitempty"`
	I18n               map[string]LessonText `json:"i18n,omitempty"`
}

// IntroIn returns the lesson's opening message in a language (English fallback).
func (l *Lesson) IntroIn(lang string) string {
	if t, ok := l.I18n[lang]; ok && strings.TrimSpace(t.Intro) != "" {
		return t.Intro
	}
	return l.Intro
}

// TutorPolicy is the course-specific teaching contract given to the AI.
type TutorPolicy struct {
	Policy          string `json:"policy"`
	EvaluatorPolicy string `json:"evaluatorPolicy"`
	// LanguagePolicy adds per-UI-language guidance, e.g. "explain mainly in Indonesian".
	LanguagePolicy map[string]string `json:"languagePolicy,omitempty"`
}

type CourseText struct {
	Title       string `json:"title"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
}

// Course is the metadata in <course>/course.json.
type Course struct {
	Slug        string      `json:"slug"`
	Title       string      `json:"title"`
	Tagline     string      `json:"tagline"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Tone        string      `json:"tone"`
	Status      string      `json:"status"` // published | coming_soon
	TargetLevel string      `json:"targetLevel"`
	LevelIDs    []string    `json:"levels"` // ordered bands; each has <id>.json
	Tutor       TutorPolicy `json:"tutor"`
	Onboarding  *Onboarding `json:"onboarding,omitempty"`
	// Languages the course content is available in; "en" is the base and
	// every other listed language must be fully translated.
	Languages []string              `json:"languages,omitempty"`
	I18n      map[string]CourseText `json:"i18n,omitempty"`
	// RecommendedAfter is where a learner should be in another course
	// before starting this one (e.g. a test-prep track after English B1).
	RecommendedAfter *LevelRef `json:"recommendedAfter,omitempty"`
	// RelatedTracks are courses suggested at the end of this roadmap
	// ("where to go next"), by slug.
	RelatedTracks []string `json:"relatedTracks,omitempty"`
}

// LevelRef points at a level of a course.
type LevelRef struct {
	Course string `json:"course"`
	Level  string `json:"level"`
}

// NodeRef points at a node of a course.
type NodeRef struct {
	Course string `json:"course"`
	NodeID string `json:"nodeId"`
}

// BaseLanguage is the language course files are written in.
const BaseLanguage = "en"

const (
	CoursePublished  = "published"
	CourseComingSoon = "coming_soon"
)

// Curriculum is an immutable, validated course roadmap.
type Curriculum struct {
	Course
	Version string
	Levels  []LevelInfo
	Nodes   []*Node // topologically sorted (stable: level, then file order)
	byID    map[string]*Node
	lessons map[string]*Lesson
	// Per-language copies of nodes and levels with text swapped in.
	localNodes  map[string]map[string]*Node
	localLevels map[string][]LevelInfo
	// successors maps a node to nodes that list it as a prerequisite.
	successors map[string][]string
	// Dir is the course folder (packages/courses/<slug>).
	Dir string
	// plans holds the hand-written lesson steps: node -> language -> plan
	// JSON (validated by the tutor package, which owns the format).
	plans map[string]map[string]json.RawMessage
}

// AuthoredPlan returns the hand-written steps of a lesson in a language.
func (c *Curriculum) AuthoredPlan(nodeID, lang string) (json.RawMessage, bool) {
	raw, ok := c.plans[nodeID][lang]
	return raw, ok
}

// SpeechKey is the normalised form of a sentence that audio is stored under.
func SpeechKey(text string) string { return strings.Join(strings.Fields(text), " ") }

// AudioPath is where a sentence's recorded audio lives in the course folder
// (audio/<sha256 of the sentence>.mp3). Lesson audio is kept in the repo so
// a deployment never has to make it again.
func (c *Curriculum) AudioPath(text string) string {
	h := sha256.Sum256([]byte(SpeechKey(text)))
	return filepath.Join(c.Dir, "audio", hex.EncodeToString(h[:])+".mp3")
}

// DropAuthoredPlans makes the AI write each learner's plan instead of using
// the hand-written steps (for when personalisation comes back, and tests).
func (c *Curriculum) DropAuthoredPlans() { c.plans = nil }

// PlanLanguages lists the languages a lesson has hand-written steps in.
func (c *Curriculum) PlanLanguages(nodeID string) []string {
	var out []string
	for l := range c.plans[nodeID] {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

func (c *Curriculum) Node(id string) (*Node, bool) {
	n, ok := c.byID[id]
	return n, ok
}

func (c *Curriculum) Lesson(nodeID string) (*Lesson, bool) {
	l, ok := c.lessons[nodeID]
	return l, ok
}

func (c *Curriculum) Successors(id string) []string { return c.successors[id] }

// LevelIndex returns the position of a level in this course's bands, or -1.
func (c *Curriculum) LevelIndex(level string) int {
	for i, l := range c.LevelIDs {
		if l == level {
			return i
		}
	}
	return -1
}

// LoadCourse reads <dir>/course.json, <dir>/<level>.json for every level and
// <dir>/lessons/*.json, and validates the result.
func LoadCourse(dir string) (*Curriculum, error) {
	var course Course
	coursePath := filepath.Join(dir, "course.json")
	raw, err := os.ReadFile(coursePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", coursePath, err)
	}
	if err := decodeStrict(raw, &course); err != nil {
		return nil, fmt.Errorf("parse %s: %w", coursePath, err)
	}
	var files []levelFile
	for _, lvl := range course.LevelIDs {
		path := filepath.Join(dir, strings.ToLower(lvl)+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var f levelFile
		if err := decodeStrict(raw, &f); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if f.Level != lvl {
			return nil, fmt.Errorf("%s: level field is %q, want %q", path, f.Level, lvl)
		}
		files = append(files, f)
	}

	lessons := map[string]*Lesson{}
	lessonPaths, err := filepath.Glob(filepath.Join(dir, "lessons", "*.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range lessonPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var l Lesson
		if err := decodeStrict(raw, &l); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if _, dup := lessons[l.NodeID]; dup {
			return nil, fmt.Errorf("duplicate lesson for node %q", l.NodeID)
		}
		lessons[l.NodeID] = &l
	}
	c, err := Build(course, files, lessons)
	if err != nil {
		return nil, fmt.Errorf("course %q: %w", course.Slug, err)
	}
	c.Dir = dir
	if c.plans, err = loadPlans(c, dir); err != nil {
		return nil, fmt.Errorf("course %q: %w", course.Slug, err)
	}
	return c, nil
}

// loadPlans reads plans/<nodeId>.json: {"en": {...}, "id": {...}}.
func loadPlans(c *Curriculum, dir string) (map[string]map[string]json.RawMessage, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "plans", "*.json"))
	if err != nil {
		return nil, err
	}
	langs := map[string]bool{BaseLanguage: true}
	for _, l := range c.Languages {
		langs[l] = true
	}
	out := map[string]map[string]json.RawMessage{}
	for _, path := range paths {
		id := strings.TrimSuffix(filepath.Base(path), ".json")
		if _, ok := c.Lesson(id); !ok {
			return nil, fmt.Errorf("%s: no lesson %q", path, id)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var byLang map[string]json.RawMessage
		if err := json.Unmarshal(raw, &byLang); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for l := range byLang {
			if !langs[l] {
				return nil, fmt.Errorf("%s: language %q isn't one of the course's", path, l)
			}
		}
		out[id] = byLang
	}
	return out, nil
}

// DisableOnboarding turns personalisation off for every course: no
// onboarding questions, goal badges, goal-based targets or tutor hints.
// The course files and stored answers are kept for when it comes back.
func (c *Catalog) DisableOnboarding() {
	for _, cur := range c.Courses {
		cur.Onboarding = nil
	}
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// Build assembles and validates a curriculum from already-parsed parts.
func Build(course Course, files []levelFile, lessons map[string]*Lesson) (*Curriculum, error) {
	c := &Curriculum{Course: course, byID: map[string]*Node{}, lessons: lessons, successors: map[string][]string{}}
	var errs []error
	errs = append(errs, validateCourse(course)...)
	levelRank := map[string]int{}
	for i, l := range course.LevelIDs {
		levelRank[l] = i
	}

	var ordered []*Node
	for _, f := range files {
		if c.Version == "" {
			c.Version = f.Version
		} else if f.Version != c.Version {
			errs = append(errs, fmt.Errorf("level %s has version %q, others have %q", f.Level, f.Version, c.Version))
		}
		c.Levels = append(c.Levels, LevelInfo{Level: f.Level, Title: f.Title, Description: f.Description})
		for _, lang := range course.Languages {
			if lang == BaseLanguage {
				continue
			}
			t := f.I18n[lang]
			if strings.TrimSpace(t.Title) == "" {
				errs = append(errs, fmt.Errorf("level %s: missing %q title", f.Level, lang))
			}
			if c.localLevels == nil {
				c.localLevels = map[string][]LevelInfo{}
			}
			c.localLevels[lang] = append(c.localLevels[lang], LevelInfo{Level: f.Level, Title: t.Title, Description: t.Description})
		}
		for i := range f.Nodes {
			n := &f.Nodes[i]
			if n.Level == "" {
				n.Level = f.Level
			}
			if n.MasteryThreshold == 0 {
				n.MasteryThreshold = DefaultMasteryThreshold
			}
			if n.PrerequisiteIDs == nil {
				n.PrerequisiteIDs = []string{}
			}
			ordered = append(ordered, n)
		}
	}

	for _, n := range ordered {
		switch {
		case n.ID == "":
			errs = append(errs, errors.New("node with empty id"))
			continue
		case c.byID[n.ID] != nil:
			errs = append(errs, fmt.Errorf("duplicate node id %q", n.ID))
			continue
		}
		c.byID[n.ID] = n
		if _, ok := levelRank[n.Level]; !ok {
			errs = append(errs, fmt.Errorf("node %q: unknown level %q", n.ID, n.Level))
		}
		if strings.TrimSpace(n.Title) == "" {
			errs = append(errs, fmt.Errorf("node %q: missing title", n.ID))
		}
		if n.ContentStatus != ContentReady && n.ContentStatus != ContentComingSoon {
			errs = append(errs, fmt.Errorf("node %q: contentStatus must be %q or %q", n.ID, ContentReady, ContentComingSoon))
		}
		if n.MasteryThreshold < 1 || n.MasteryThreshold > 100 {
			errs = append(errs, fmt.Errorf("node %q: masteryThreshold out of range", n.ID))
		}
		if len(n.Objectives) == 0 {
			errs = append(errs, fmt.Errorf("node %q: needs at least one objective", n.ID))
		}
		seenObj := map[string]bool{}
		for _, o := range n.Objectives {
			if o.ID == "" || strings.TrimSpace(o.Text) == "" {
				errs = append(errs, fmt.Errorf("node %q: objective needs id and text", n.ID))
			}
			if seenObj[o.ID] {
				errs = append(errs, fmt.Errorf("node %q: duplicate objective id %q", n.ID, o.ID))
			}
			seenObj[o.ID] = true
		}
	}

	for _, n := range ordered {
		if c.byID[n.ID] != n {
			continue
		}
		for _, p := range n.PrerequisiteIDs {
			pre, ok := c.byID[p]
			if !ok {
				errs = append(errs, fmt.Errorf("node %q: unknown prerequisite %q", n.ID, p))
				continue
			}
			if p == n.ID {
				errs = append(errs, fmt.Errorf("node %q: lists itself as prerequisite", n.ID))
				continue
			}
			if levelRank[pre.Level] > levelRank[n.Level] {
				errs = append(errs, fmt.Errorf("node %q (%s): prerequisite %q is a higher level (%s)", n.ID, n.Level, p, pre.Level))
			}
			c.successors[p] = append(c.successors[p], n.ID)
		}
	}

	errs = append(errs, validateLessons(c)...)
	errs = append(errs, validateTranslations(c)...)
	errs = append(errs, validateOnboarding(c)...)
	errs = append(errs, validateMaterial(c)...)

	if len(errs) == 0 {
		sorted, err := topoSort(ordered, c.byID)
		if err != nil {
			errs = append(errs, err)
		} else {
			c.Nodes = sorted
		}
	}

	// A published course needs somewhere to start; a coming-soon course
	// can be a roadmap of lessons that aren't written yet.
	if len(errs) == 0 && c.Status == CoursePublished {
		errs = append(errs, validateReachability(c)...)
	}
	if len(errs) > 0 {
		return nil, &ValidationError{Errs: errs}
	}
	c.buildLocalized()
	return c, nil
}

func validateLessons(c *Curriculum) []error {
	var errs []error
	for id, l := range c.lessons {
		n, ok := c.byID[id]
		if !ok {
			errs = append(errs, fmt.Errorf("lesson for unknown node %q", id))
			continue
		}
		if !n.Ready() {
			errs = append(errs, fmt.Errorf("lesson %q exists but node is %q; mark it ready or remove the lesson", id, n.ContentStatus))
		}
		if strings.TrimSpace(l.Intro) == "" {
			errs = append(errs, fmt.Errorf("lesson %q: missing intro", id))
		}
		objIDs := map[string]bool{}
		for _, o := range n.Objectives {
			objIDs[o.ID] = true
		}
		covered := map[string]bool{}
		itemIDs := map[string]bool{}
		for _, it := range l.Assessment {
			if it.ID == "" || strings.TrimSpace(it.Prompt) == "" {
				errs = append(errs, fmt.Errorf("lesson %q: assessment item needs id and prompt", id))
			}
			if itemIDs[it.ID] {
				errs = append(errs, fmt.Errorf("lesson %q: duplicate assessment item %q", id, it.ID))
			}
			itemIDs[it.ID] = true
			if !objIDs[it.ObjectiveID] {
				errs = append(errs, fmt.Errorf("lesson %q: item %q targets unknown objective %q", id, it.ID, it.ObjectiveID))
			}
			if len(it.AcceptableAnswers) == 0 {
				errs = append(errs, fmt.Errorf("lesson %q: item %q needs at least one acceptable answer", id, it.ID))
			}
			covered[it.ObjectiveID] = true
		}
		for _, p := range append(append([]PracticePrompt{}, l.GuidedPractice...), l.IndependentPrompt...) {
			if !objIDs[p.ObjectiveID] {
				errs = append(errs, fmt.Errorf("lesson %q: practice prompt targets unknown objective %q", id, p.ObjectiveID))
			}
		}
		for _, o := range n.Objectives {
			if !covered[o.ID] {
				errs = append(errs, fmt.Errorf("lesson %q: objective %q has no assessment item", id, o.ID))
			}
		}
		if len(l.GuidedPractice) == 0 || len(l.IndependentPrompt) == 0 {
			errs = append(errs, fmt.Errorf("lesson %q: needs guided and independent practice prompts", id))
		}
	}
	for _, n := range c.byID {
		if n.Ready() && c.lessons[n.ID] == nil {
			errs = append(errs, fmt.Errorf("node %q is ready but has no lesson file", n.ID))
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// topoSort orders nodes with Kahn's algorithm, keeping file order among peers.
func topoSort(ordered []*Node, byID map[string]*Node) ([]*Node, error) {
	indeg := map[string]int{}
	for _, n := range ordered {
		indeg[n.ID] = len(n.PrerequisiteIDs)
	}
	pos := map[string]int{}
	for i, n := range ordered {
		pos[n.ID] = i
	}
	succ := map[string][]string{}
	for _, n := range ordered {
		for _, p := range n.PrerequisiteIDs {
			succ[p] = append(succ[p], n.ID)
		}
	}
	var queue []string
	for _, n := range ordered {
		if indeg[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}
	var out []*Node
	for len(queue) > 0 {
		sort.Slice(queue, func(i, j int) bool { return pos[queue[i]] < pos[queue[j]] })
		id := queue[0]
		queue = queue[1:]
		out = append(out, byID[id])
		for _, s := range succ[id] {
			indeg[s]--
			if indeg[s] == 0 {
				queue = append(queue, s)
			}
		}
	}
	if len(out) != len(ordered) {
		var cyc []string
		for _, n := range ordered {
			if indeg[n.ID] > 0 {
				cyc = append(cyc, n.ID)
			}
		}
		sort.Strings(cyc)
		return nil, fmt.Errorf("prerequisite cycle involving: %s", strings.Join(cyc, ", "))
	}
	return out, nil
}

// validateReachability ensures there is at least one ready root, so a fresh
// learner can start somewhere.
func validateReachability(c *Curriculum) []error {
	for _, n := range c.Nodes {
		if len(n.PrerequisiteIDs) == 0 && n.Ready() {
			return nil
		}
	}
	return []error{errors.New("no ready root node: a new learner could not start any lesson")}
}

type ValidationError struct{ Errs []error }

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Errs))
	for i, err := range e.Errs {
		msgs[i] = "  - " + err.Error()
	}
	return fmt.Sprintf("curriculum invalid (%d problems):\n%s", len(e.Errs), strings.Join(msgs, "\n"))
}

// validateTranslations requires every non-base language a course declares
// to be complete, so a learner never sees a half-translated roadmap.
func validateTranslations(c *Curriculum) []error {
	var errs []error
	hasBase := len(c.Languages) == 0
	for _, lang := range c.Languages {
		if lang == BaseLanguage {
			hasBase = true
			continue
		}
		ct := c.I18n[lang]
		if strings.TrimSpace(ct.Title) == "" || strings.TrimSpace(ct.Tagline) == "" || strings.TrimSpace(ct.Description) == "" {
			errs = append(errs, fmt.Errorf("course: missing %q title, tagline or description", lang))
		}
		if strings.TrimSpace(c.Tutor.LanguagePolicy[lang]) == "" {
			errs = append(errs, fmt.Errorf("course: missing tutor.languagePolicy for %q", lang))
		}
		for _, n := range c.byID {
			t, ok := n.I18n[lang]
			if !ok || strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.Summary) == "" {
				errs = append(errs, fmt.Errorf("node %q: missing %q title or summary", n.ID, lang))
				continue
			}
			for _, o := range n.Objectives {
				if strings.TrimSpace(t.Objectives[o.ID]) == "" {
					errs = append(errs, fmt.Errorf("node %q: missing %q text for objective %q", n.ID, lang, o.ID))
				}
			}
		}
		for id, l := range c.lessons {
			if strings.TrimSpace(l.I18n[lang].Intro) == "" {
				errs = append(errs, fmt.Errorf("lesson %q: missing %q intro", id, lang))
			}
		}
	}
	if !hasBase {
		errs = append(errs, fmt.Errorf("course languages must include %q", BaseLanguage))
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// buildLocalized precomputes per-language node copies (and base copies
// without the i18n payload) so views never mutate shared nodes.
func (c *Curriculum) buildLocalized() {
	c.localNodes = map[string]map[string]*Node{}
	langs := append([]string{BaseLanguage}, c.Languages...)
	for _, lang := range langs {
		if c.localNodes[lang] != nil {
			continue
		}
		m := map[string]*Node{}
		for _, n := range c.Nodes {
			cp := *n
			cp.I18n = nil
			if t, ok := n.I18n[lang]; ok && lang != BaseLanguage {
				cp.Title, cp.Summary = t.Title, t.Summary
				cp.Objectives = make([]Objective, len(n.Objectives))
				for i, o := range n.Objectives {
					cp.Objectives[i] = Objective{ID: o.ID, Text: t.Objectives[o.ID]}
				}
			}
			m[n.ID] = &cp
		}
		c.localNodes[lang] = m
	}
}

// Lang returns lang if the course supports it, otherwise the base language.
func (c *Curriculum) Lang(lang string) string {
	for _, l := range c.Languages {
		if l == lang {
			return lang
		}
	}
	return BaseLanguage
}

// LocalNode returns the node with titles, summary and objectives in lang.
func (c *Curriculum) LocalNode(id, lang string) *Node {
	return c.localNodes[c.Lang(lang)][id]
}

// LocalLevels returns level titles and descriptions in lang.
func (c *Curriculum) LocalLevels(lang string) []LevelInfo {
	if l, ok := c.localLevels[c.Lang(lang)]; ok {
		return l
	}
	return c.Levels
}

// Text returns the course title, tagline and description in lang.
func (c *Curriculum) Text(lang string) CourseText {
	if t, ok := c.I18n[c.Lang(lang)]; ok {
		if t.Category == "" {
			t.Category = c.Category
		}
		return t
	}
	return CourseText{c.Title, c.Tagline, c.Description, c.Category}
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

func validateCourse(c Course) []error {
	var errs []error
	if !slugRe.MatchString(c.Slug) {
		errs = append(errs, fmt.Errorf("course slug %q must be lowercase letters, digits and dashes", c.Slug))
	}
	if strings.TrimSpace(c.Title) == "" {
		errs = append(errs, errors.New("course needs a title"))
	}
	if c.Status != CoursePublished && c.Status != CourseComingSoon {
		errs = append(errs, fmt.Errorf("course status must be %q or %q", CoursePublished, CourseComingSoon))
	}
	if len(c.LevelIDs) == 0 {
		errs = append(errs, errors.New("course needs at least one level"))
	}
	seen := map[string]bool{}
	target := false
	for _, l := range c.LevelIDs {
		if l == "" || seen[l] {
			errs = append(errs, fmt.Errorf("level ids must be unique and non-empty (got %q)", l))
		}
		seen[l] = true
		target = target || l == c.TargetLevel
	}
	if !target {
		errs = append(errs, fmt.Errorf("targetLevel %q is not one of the course levels", c.TargetLevel))
	}
	if strings.TrimSpace(c.Tutor.Policy) == "" || strings.TrimSpace(c.Tutor.EvaluatorPolicy) == "" {
		errs = append(errs, errors.New("course needs tutor.policy and tutor.evaluatorPolicy"))
	}
	return errs
}

// Catalog is every course under the courses directory, in title order.
type Catalog struct {
	Courses []*Curriculum
	bySlug  map[string]*Curriculum
}

func (c *Catalog) Course(slug string) (*Curriculum, bool) {
	cur, ok := c.bySlug[slug]
	return cur, ok
}

// LoadCatalog loads and validates every <root>/<slug>/course.json.
func LoadCatalog(root string) (*Catalog, error) {
	dirs, err := filepath.Glob(filepath.Join(root, "*", "course.json"))
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no courses found in %s (expected <slug>/course.json)", root)
	}
	cat := &Catalog{bySlug: map[string]*Curriculum{}}
	for _, p := range dirs {
		dir := filepath.Dir(p)
		c, err := LoadCourse(dir)
		if err != nil {
			return nil, err
		}
		if filepath.Base(dir) != c.Slug {
			return nil, fmt.Errorf("course in folder %q has slug %q; they must match", filepath.Base(dir), c.Slug)
		}
		if cat.bySlug[c.Slug] != nil {
			return nil, fmt.Errorf("duplicate course slug %q", c.Slug)
		}
		cat.bySlug[c.Slug] = c
		cat.Courses = append(cat.Courses, c)
	}
	sort.Slice(cat.Courses, func(i, j int) bool { return cat.Courses[i].Title < cat.Courses[j].Title })
	if err := cat.validateLinks(); err != nil {
		return nil, err
	}
	return cat, nil
}

// validateLinks checks references between courses: related tracks,
// recommended levels and node foundations must exist.
func (c *Catalog) validateLinks() error {
	var errs []error
	for _, cur := range c.Courses {
		for _, slug := range cur.RelatedTracks {
			if _, ok := c.bySlug[slug]; !ok || slug == cur.Slug {
				errs = append(errs, fmt.Errorf("course %q: related track %q isn't another course", cur.Slug, slug))
			}
		}
		if r := cur.RecommendedAfter; r != nil {
			other, ok := c.bySlug[r.Course]
			if !ok || !slices.Contains(other.LevelIDs, r.Level) {
				errs = append(errs, fmt.Errorf("course %q: recommendedAfter %s/%s doesn't exist", cur.Slug, r.Course, r.Level))
			}
		}
		for _, n := range cur.Nodes {
			for _, f := range n.Foundations {
				other, ok := c.bySlug[f.Course]
				if !ok {
					errs = append(errs, fmt.Errorf("course %q node %q: foundation course %q doesn't exist", cur.Slug, n.ID, f.Course))
					continue
				}
				if _, ok := other.Node(f.NodeID); !ok {
					errs = append(errs, fmt.Errorf("course %q node %q: foundation %s/%s doesn't exist", cur.Slug, n.ID, f.Course, f.NodeID))
				}
			}
		}
	}
	return errors.Join(errs...)
}

// NewCatalog builds a catalog from already-loaded courses (tests).
func NewCatalog(courses ...*Curriculum) *Catalog {
	cat := &Catalog{bySlug: map[string]*Curriculum{}}
	for _, c := range courses {
		cat.bySlug[c.Slug] = c
		cat.Courses = append(cat.Courses, c)
	}
	return cat
}
