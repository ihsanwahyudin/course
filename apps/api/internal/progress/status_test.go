package progress

import (
	"testing"

	"fluentpath/api/internal/curriculum"
)

func testCurriculum(t *testing.T) *curriculum.Curriculum {
	t.Helper()
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDeriveFreshLearner(t *testing.T) {
	c := testCurriculum(t)
	st := Derive(c, nil)
	if st["a0-alphabet-sounds"] != StatusAvailable {
		t.Errorf("root should be available, got %s", st["a0-alphabet-sounds"])
	}
	if st["a0-pronouns"] != StatusLocked {
		t.Errorf("pronouns should be locked, got %s", st["a0-pronouns"])
	}
	if st["c2-stylistic-nuance"] != StatusComingSoon {
		t.Errorf("unbuilt node should be coming soon, got %s", st["c2-stylistic-nuance"])
	}
	if n := NextSuggested(c, st, nil); n == nil || n.ID != "a0-alphabet-sounds" {
		t.Errorf("next suggested = %v", n)
	}
}

func TestUnlockAfterMastery(t *testing.T) {
	c := testCurriculum(t)
	before := Derive(c, map[string]Stored{"a0-alphabet-sounds": {Status: "in_progress"}})
	after := Derive(c, map[string]Stored{"a0-alphabet-sounds": {Status: "mastered"}})
	got := NewlyUnlocked(c, before, after)
	want := map[string]bool{"a0-pronouns": true, "a0-common-vocabulary": true}
	if len(got) != len(want) {
		t.Fatalf("unlocked %v, want %v", got, want)
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected unlock %s", id)
		}
	}
	// a0-basic-sentences needs BOTH pronouns and vocabulary.
	partial := Derive(c, map[string]Stored{"a0-alphabet-sounds": {Status: "mastered"}, "a0-pronouns": {Status: "mastered"}})
	if partial["a0-basic-sentences"] != StatusLocked {
		t.Errorf("basic sentences should stay locked until all prerequisites are mastered")
	}
}

func TestCurrentLevel(t *testing.T) {
	c := testCurriculum(t)
	stored := map[string]Stored{}
	for _, id := range []string{"a0-alphabet-sounds", "a0-pronouns", "a0-common-vocabulary", "a0-basic-sentences"} {
		stored[id] = Stored{Status: "mastered"}
	}
	if lvl := CurrentLevel(c, Derive(c, stored)); lvl != "A1" {
		t.Errorf("current level = %s, want A1", lvl)
	}
}

func TestNextSuggestedPrefersGoalNodes(t *testing.T) {
	c := testCurriculum(t)
	st := Derive(c, map[string]Stored{"a0-alphabet-sounds": {Status: "mastered"}})
	vocab := func(n *curriculum.Node) bool { return n.ID == "a0-common-vocabulary" }
	if n := NextSuggested(c, st, nil); n == nil || n.ID != "a0-pronouns" {
		t.Fatalf("without a preference, curriculum order: got %v", n)
	}
	if n := NextSuggested(c, st, vocab); n == nil || n.ID != "a0-common-vocabulary" {
		t.Fatalf("preferred available node first: got %v", n)
	}
	locked := func(n *curriculum.Node) bool { return n.ID == "a1-to-be" }
	if n := NextSuggested(c, st, locked); n == nil || n.ID != "a0-pronouns" {
		t.Fatalf("a preferred but locked node must not be suggested: got %v", n)
	}
	st = Derive(c, map[string]Stored{"a0-alphabet-sounds": {Status: "mastered"}, "a0-pronouns": {Status: "in_progress"}})
	if n := NextSuggested(c, st, vocab); n == nil || n.ID != "a0-pronouns" {
		t.Fatalf("an in-progress node still comes first: got %v", n)
	}
}
