package curriculum

import (
	"strings"
	"testing"
)

func TestLoadRepositoryCurriculum(t *testing.T) {
	c, err := LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	ready := 0
	for _, n := range c.Nodes {
		if n.Ready() {
			ready++
		}
	}
	if ready < 5 {
		t.Fatalf("want at least 5 ready lessons, got %d", ready)
	}
	if len(c.Levels) != len(c.LevelIDs) || c.Slug != "english" || c.Tutor.Policy == "" {
		t.Fatalf("course metadata not loaded: %+v", c.Course)
	}
	// Topological order: every prerequisite appears before its dependant.
	pos := map[string]int{}
	for i, n := range c.Nodes {
		pos[n.ID] = i
	}
	for _, n := range c.Nodes {
		for _, p := range n.PrerequisiteIDs {
			if pos[p] >= pos[n.ID] {
				t.Errorf("%s appears before its prerequisite %s", n.ID, p)
			}
		}
	}
}

func obj(id string) []Objective { return []Objective{{ID: id, Text: id}} }

func readyLesson(nodeID, objID string) *Lesson {
	p := []PracticePrompt{{ObjectiveID: objID, Prompt: "p", Expected: []string{"x"}}}
	return &Lesson{NodeID: nodeID, Intro: "hi", GuidedPractice: p, IndependentPrompt: p,
		Assessment: []AssessmentItem{{ID: nodeID + "-1", ObjectiveID: objID, Prompt: "q", AcceptableAnswers: []string{"x"}}},
		Material:   testMaterial()}
}

func testMaterial() []Section {
	text := func(id string) Section {
		return Section{ID: id, Title: id, Blocks: []Block{{Type: BlockText, Text: "t", I18n: map[string]BlockI18n{"id": {Text: "t"}}}},
			I18n: map[string]SectionText{"id": {Title: id}}}
	}
	return []Section{text("one"), text("two"), text("three")}
}

var testCourse = Course{
	Slug: "test", Title: "Test", Status: CoursePublished, TargetLevel: "A0", LevelIDs: []string{"A0"},
	Tutor: TutorPolicy{Policy: "teach", EvaluatorPolicy: "grade"},
}

func build(nodes []Node, lessons map[string]*Lesson) error {
	_, err := Build(testCourse, []levelFile{{Version: "1", Level: "A0", Nodes: nodes}}, lessons)
	return err
}

func wantErr(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), substr) {
		t.Fatalf("want error containing %q, got %v", substr, err)
	}
}

func TestValidatorRejectsMissingPrerequisite(t *testing.T) {
	err := build([]Node{
		{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentReady},
		{ID: "b", Title: "B", Objectives: obj("o"), ContentStatus: ContentComingSoon, PrerequisiteIDs: []string{"ghost"}},
	}, map[string]*Lesson{"a": readyLesson("a", "o")})
	wantErr(t, err, `unknown prerequisite "ghost"`)
}

func TestValidatorRejectsCycle(t *testing.T) {
	err := build([]Node{
		{ID: "root", Title: "R", Objectives: obj("o"), ContentStatus: ContentReady},
		{ID: "x", Title: "X", Objectives: obj("o"), ContentStatus: ContentComingSoon, PrerequisiteIDs: []string{"y"}},
		{ID: "y", Title: "Y", Objectives: obj("o"), ContentStatus: ContentComingSoon, PrerequisiteIDs: []string{"x"}},
	}, map[string]*Lesson{"root": readyLesson("root", "o")})
	wantErr(t, err, "cycle involving: x, y")
}

func TestValidatorRejectsDuplicateIDs(t *testing.T) {
	err := build([]Node{
		{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentReady},
		{ID: "a", Title: "A2", Objectives: obj("o"), ContentStatus: ContentComingSoon},
	}, map[string]*Lesson{"a": readyLesson("a", "o")})
	wantErr(t, err, `duplicate node id "a"`)
}

func TestValidatorRequiresLessonForReadyNode(t *testing.T) {
	err := build([]Node{{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentReady}}, nil)
	wantErr(t, err, `"a" is ready but has no lesson file`)
}

func TestValidatorRequiresAssessmentCoverage(t *testing.T) {
	l := readyLesson("a", "o1")
	err := build([]Node{{ID: "a", Title: "A", Objectives: []Objective{{ID: "o1", Text: "one"}, {ID: "o2", Text: "two"}}, ContentStatus: ContentReady}},
		map[string]*Lesson{"a": l})
	wantErr(t, err, `objective "o2" has no assessment item`)
}

func TestValidatorRequiresReadyRoot(t *testing.T) {
	err := build([]Node{{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentComingSoon}}, nil)
	wantErr(t, err, "no ready root node")
}

func TestValidatorRejectsBadCourseMetadata(t *testing.T) {
	bad := testCourse
	bad.Slug, bad.TargetLevel, bad.Tutor.Policy = "Bad Slug", "Z9", ""
	_, err := Build(bad, []levelFile{{Version: "1", Level: "A0", Nodes: []Node{{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentReady}}}},
		map[string]*Lesson{"a": readyLesson("a", "o")})
	for _, want := range []string{"course slug", "targetLevel", "tutor.policy"} {
		wantErr(t, err, want)
	}
}

func TestLoadCatalog(t *testing.T) {
	cat, err := LoadCatalog("../../../../packages/courses")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Course("english"); !ok || len(cat.Courses) == 0 {
		t.Fatalf("english course missing from catalog")
	}
	if _, err := LoadCatalog(t.TempDir()); err == nil {
		t.Fatal("empty courses dir should fail")
	}
}

func TestValidatorRequiresCompleteTranslations(t *testing.T) {
	course := testCourse
	course.Languages = []string{"en", "id"}
	course.I18n = map[string]CourseText{"id": {Title: "Tes", Tagline: "t", Description: "d"}}
	course.Tutor.LanguagePolicy = map[string]string{"id": "Jelaskan dalam Bahasa Indonesia."}
	node := Node{ID: "a", Title: "A", Objectives: []Objective{{ID: "o1", Text: "one"}, {ID: "o2", Text: "two"}}, ContentStatus: ContentReady,
		I18n: map[string]NodeText{"id": {Title: "A-id", Summary: "ringkas", Objectives: map[string]string{"o1": "satu"}}}}
	l := readyLesson("a", "o1")
	l.Assessment = append(l.Assessment, AssessmentItem{ID: "a-2", ObjectiveID: "o2", Prompt: "q", AcceptableAnswers: []string{"x"}})
	_, err := Build(course, []levelFile{{Version: "1", Level: "A0", Nodes: []Node{node}}}, map[string]*Lesson{"a": l})
	for _, want := range []string{`level A0: missing "id" title`, `node "a": missing "id" text for objective "o2"`, `lesson "a": missing "id" intro`} {
		wantErr(t, err, want)
	}
}

func TestLocalizedViews(t *testing.T) {
	c, err := LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	id := c.LocalNode("a0-pronouns", "id")
	en := c.LocalNode("a0-pronouns", "en")
	if id.Title != "Kata Ganti Orang" || en.Title != "Personal Pronouns" || id.I18n != nil || en.I18n != nil {
		t.Fatalf("localized nodes wrong: %q %q", id.Title, en.Title)
	}
	if id.Objectives[0].ID != en.Objectives[0].ID || id.Objectives[0].Text == en.Objectives[0].Text {
		t.Fatalf("objective ids must stay, text must change: %+v", id.Objectives[0])
	}
	if c.LocalNode("a0-pronouns", "fr").Title != "Personal Pronouns" || c.Lang("fr") != "en" {
		t.Fatal("unsupported languages fall back to English")
	}
	if c.Text("id").Title != "Bahasa Inggris" || c.LocalLevels("id")[0].Title != "Dasar" {
		t.Fatal("course/level text not localized")
	}
	l, _ := c.Lesson("a0-pronouns")
	if l.IntroIn("id") == l.Intro || l.IntroIn("fr") != l.Intro {
		t.Fatal("lesson intro localization wrong")
	}
}

func TestComingSoonCourseNeedsNoReadyLesson(t *testing.T) {
	soon := testCourse
	soon.Status = CourseComingSoon
	_, err := Build(soon, []levelFile{{Version: "1", Level: "A0", Nodes: []Node{{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentComingSoon}}}}, nil)
	if err != nil {
		t.Fatalf("a coming-soon roadmap of unwritten lessons is fine: %v", err)
	}
}

func TestCatalogValidatesLinksBetweenCourses(t *testing.T) {
	english, err := LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	english.RelatedTracks = nil // its real tracks aren't in this small catalog
	track := testCourse
	track.Slug, track.Status = "track", CourseComingSoon
	mk := func(rec *LevelRef, related []string, f []NodeRef) *Catalog {
		c := track
		c.RecommendedAfter, c.RelatedTracks = rec, related
		cur, err := Build(c, []levelFile{{Version: "1", Level: "A0", Nodes: []Node{{ID: "a", Title: "A", Objectives: obj("o"), ContentStatus: ContentComingSoon, Foundations: f}}}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return NewCatalog(english, cur)
	}
	ok := mk(&LevelRef{"english", "B1"}, []string{"english"}, []NodeRef{{"english", "b2-passive-voice"}})
	if err := ok.validateLinks(); err != nil {
		t.Fatalf("valid links rejected: %v", err)
	}
	for name, cat := range map[string]*Catalog{
		"unknown level":      mk(&LevelRef{"english", "Z9"}, nil, nil),
		"unknown track":      mk(nil, []string{"klingon"}, nil),
		"self track":         mk(nil, []string{"track"}, nil),
		"unknown foundation": mk(nil, nil, []NodeRef{{"english", "ghost"}}),
		"unknown course":     mk(nil, nil, []NodeRef{{"klingon", "x"}}),
	} {
		if err := cat.validateLinks(); err == nil {
			t.Errorf("%s: should be rejected", name)
		}
	}
}
