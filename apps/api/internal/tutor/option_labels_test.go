package tutor

import (
	"encoding/json"
	"slices"
	"strconv"
	"testing"

	"fluentpath/api/internal/curriculum"
)

func TestOptionLabelsPreserveQuestionsAndHints(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	en, _, err := AuthoredPlan(c, "a0-sounds-indonesian", "en")
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := AuthoredPlan(c, "a0-sounds-indonesian", "id")
	if err != nil {
		t.Fatal(err)
	}
	q := *id.Steps[3].Question
	if !SameSteps(en, id) || q.Answer != 1 || q.OptionsForDisplay()[1] != "Letakkan ujung lidah di antara gigi untuk bunyi th" {
		t.Fatalf("translations must preserve the answer and lesson version: %+v", q)
	}
	if !slices.Equal(id.Steps[1].Question.OptionsForDisplay(), en.Steps[1].Question.Options) {
		t.Fatal("ship / sheep are English targets, not labels to translate")
	}
	for _, label := range []string{q.Options[q.Answer], q.DisplayOptions[q.Answer]} {
		if !LeaksAnswer("Jawabannya: "+label+".", q) {
			t.Fatalf("hint leaked %q", label)
		}
	}
	for _, labels := range [][]string{{"satu"}, {"satu", "", "tiga"}, {"satu", "SATU", "tiga"}} {
		bad := q
		bad.DisplayOptions = labels
		if _, err := validateQuestion(bad, map[string]bool{q.ObjectiveID: true}, 400); err == nil {
			t.Fatalf("accepted invalid labels: %v", labels)
		}
	}
	q.Type = QuestionWordOrder
	if _, err := checkBasics(q, map[string]bool{q.ObjectiveID: true}, 400); err == nil {
		t.Fatal("word-order cannot have option labels")
	}
}

func TestTranslatedCheckpointLabelsFollowShufflingAndGrading(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range []string{"a1-pron-endings", "a1-pron-word-stress"} {
		l, _ := c.Lesson(node)
		ds := AuthoredDrafts(l, "id")
		items, err := FromDrafts(ds)
		if err != nil {
			t.Fatal(err)
		}
		for i := range items {
			items[i].ID = "q" + strconv.Itoa(i+1)
		}
		answers := map[string]string{}
		fixed := map[string]int{}
		gradeRows := []map[string]any{}
		translated := 0
		for i, it := range items {
			if len(ds[i].DisplayOptions) == 0 {
				continue
			}
			translated++
			labels := map[string]string{}
			for j, canonical := range ds[i].Options {
				labels[canonical] = ds[i].DisplayOptions[j]
			}
			for j, canonical := range it.Options {
				if it.OptionsForDisplay()[j] != labels[canonical] {
					t.Fatalf("label shuffled away from %q", canonical)
				}
			}
			if it.Key() != labels[ds[i].CorrectOption] {
				t.Fatalf("wrong translated key: %s", it.Key())
			}
			answers[it.ID] = strconv.Itoa(it.Answer)
			if score, ok := FixedVerdict(it, answers[it.ID], nil); !ok || score != 100 {
				t.Fatal("translated choice did not earn full credit")
			}
			fixed[it.ID] = 100
			gradeRows = append(gradeRows, map[string]any{"id": it.ID, "correctAnswer": it.Key(), "explanation": "Benar.", "score": 100})
		}
		if translated != 2 {
			t.Fatalf("expected two translated checkpoint items in %s", node)
		}
		// Review text and options must agree with what the learner chose.
		selected := []CheckItem{}
		for _, it := range items {
			if _, ok := answers[it.ID]; ok {
				selected = append(selected, it)
			}
		}
		raw, _ := json.Marshal(map[string]any{"items": gradeRows})
		review, _, _, err := ApplyGrade(raw, selected, answers, fixed, "id")
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range review.Items {
			if r.Answer != selected[i].Key() || r.CorrectAnswer != r.Answer || r.Score != 100 || !slices.Equal(r.Options, selected[i].OptionsForDisplay()) {
				t.Fatalf("review lost labels: %+v", r)
			}
		}
	}
}

func TestReviewedOptionLabelsAcrossAllLevels(t *testing.T) {
	c, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	plans, checks, material := 0, 0, 0
	byLevel := map[string]int{}
	for _, n := range c.Nodes {
		if !n.Ready() || n.LevelCheck() {
			continue
		}
		en, ok, err := AuthoredPlan(c, n.ID, "en")
		if err != nil || !ok {
			t.Fatalf("%s: %v", n.ID, err)
		}
		id, ok, err := AuthoredPlan(c, n.ID, "id")
		if err != nil || !ok || !SameSteps(en, id) {
			t.Fatalf("%s lost EN/ID grading parity: %v", n.ID, err)
		}
		for _, st := range id.Steps {
			q := st.Question
			if q == nil || len(q.DisplayOptions) == 0 {
				continue
			}
			plans++
			byLevel[n.Level]++
			if q.DisplayOptions[q.Answer] != q.OptionsForDisplay()[q.Answer] {
				t.Fatalf("%s: wrong displayed key", n.ID)
			}
			legacy := *q
			legacy.DisplayOptions = nil
			if !sameQuestion(legacy, *q) {
				t.Fatalf("%s: translated labels must not restart a lesson", n.ID)
			}
		}
		l, _ := c.Lesson(n.ID)
		ds := AuthoredDrafts(l, "id")
		items, err := FromDrafts(ds)
		if err != nil {
			t.Fatal(err)
		}
		objectives := map[string]bool{}
		for _, o := range n.Objectives {
			objectives[o.ID] = true
		}
		items, err = ValidateCheckpoint(items, objectives, nil)
		if err != nil {
			t.Fatalf("%s: %v", n.ID, err)
		}
		for i, it := range items {
			if len(it.DisplayOptions) == 0 {
				continue
			}
			checks++
			byLevel[n.Level]++
			answer := strconv.Itoa(it.Answer)
			if score, ok := FixedVerdict(it, answer, nil); !ok || score != 100 || AnswerText(it, answer) != it.Key() {
				t.Fatalf("%s: translated checkpoint is graded incorrectly", n.ID)
			}
			index := slices.Index(ds[i].Options, ds[i].CorrectOption)
			if index < 0 || it.Key() != ds[i].DisplayOptions[index] {
				t.Fatalf("%s: shuffled checkpoint key lost its label", n.ID)
			}
		}
		enMat, idMat := c.MaterialIn(n.ID, "en"), c.MaterialIn(n.ID, "id")
		for i, sec := range l.Material {
			for j, b := range sec.Blocks {
				labels := b.I18n["id"].Options
				if len(labels) == 0 {
					continue
				}
				material++
				byLevel[n.Level]++
				enBlock, idBlock := enMat.Sections[i].Blocks[j], idMat.Sections[i].Blocks[j]
				if !slices.Equal(enBlock.Options, b.Options) || !slices.Equal(idBlock.Options, labels) || *idBlock.Answer != *b.Answer {
					t.Fatalf("%s/%s: material translation changed options or key", n.ID, sec.ID)
				}
			}
		}
	}
	if plans != 115 || checks != 7 || material != 131 {
		t.Fatalf("reviewed translation coverage changed: plans=%d checkpoint=%d material=%d", plans, checks, material)
	}
	for level, want := range map[string]int{"A0": 10, "A1": 35, "A2": 34, "B1": 32, "B2": 39, "C1": 43, "C2": 60} {
		if byLevel[level] != want {
			t.Errorf("%s: translated=%d, reviewed=%d", level, byLevel[level], want)
		}
	}
}
