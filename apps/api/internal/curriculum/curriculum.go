// Package curriculum loads and validates the versioned roadmap JSON in
// packages/curriculum. The curriculum is the authoritative, read-only source of
// nodes, prerequisites, objectives and lesson content; nothing at runtime (and
// in particular no AI output) is allowed to change it.
package curriculum

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Levels lists the roadmap bands in order. A0 is an app-specific foundation
// track; A1–C2 follow CEFR-inspired can-do descriptors.
var Levels = []string{"A0", "A1", "A2", "B1", "B2", "C1", "C2"}

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
}

func (n *Node) Ready() bool { return n.ContentStatus == ContentReady }

type LevelInfo struct {
	Level       string `json:"level"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type levelFile struct {
	Version     string `json:"version"`
	Level       string `json:"level"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Nodes       []Node `json:"nodes"`
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
}

// Curriculum is an immutable, validated roadmap.
type Curriculum struct {
	Version string
	Levels  []LevelInfo
	Nodes   []*Node // topologically sorted (stable: level, then file order)
	byID    map[string]*Node
	lessons map[string]*Lesson
	// successors maps a node to nodes that list it as a prerequisite.
	successors map[string][]string
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

// Load reads <dir>/<level>.json for every level plus <dir>/lessons/*.json and
// validates the result.
func Load(dir string) (*Curriculum, error) {
	var files []levelFile
	for _, lvl := range Levels {
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
	return Build(files, lessons)
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// Build assembles and validates a curriculum from already-parsed parts.
func Build(files []levelFile, lessons map[string]*Lesson) (*Curriculum, error) {
	c := &Curriculum{byID: map[string]*Node{}, lessons: lessons, successors: map[string][]string{}}
	var errs []error
	levelRank := map[string]int{}
	for i, l := range Levels {
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

	if len(errs) == 0 {
		sorted, err := topoSort(ordered, c.byID)
		if err != nil {
			errs = append(errs, err)
		} else {
			c.Nodes = sorted
		}
	}

	if len(errs) == 0 {
		errs = append(errs, validateReachability(c)...)
	}
	if len(errs) > 0 {
		return nil, &ValidationError{Errs: errs}
	}
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

// LevelIndex returns the position of a level in Levels, or -1.
func LevelIndex(level string) int {
	for i, l := range Levels {
		if l == level {
			return i
		}
	}
	return -1
}
