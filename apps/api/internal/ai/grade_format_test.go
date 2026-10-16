package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"fluentpath/api/internal/tutor"
)

func TestGradeFormattingPolicyAndFallback(t *testing.T) {
	in, _ := lessonInput(t, "c2-editing", "id")
	q := tutor.CheckItem{ID: "date", Question: tutor.Question{Type: tutor.QuestionText, ObjectiveID: in.Node.Objectives[0].ID, Prompt: "House style: day + month in words + year, with no comma."}, Accepted: []string{"3 october 2026"}}
	task := gradeTask(GradeInput{LessonInput: in, Items: []tutor.CheckItem{q}})
	for _, want := range []string{`Indonesian "dan"`, "unless order is tested", "UNLESS the learner-visible prompt explicitly tests"} {
		if !strings.Contains(task, want) {
			t.Errorf("grading policy missing %q", want)
		}
	}
	for _, tt := range []struct {
		answer string
		want   int
	}{{"3 October 2026", 100}, {"3, October, 2026", 0}, {"October 3, 2026", 0}} {
		r, err := (&Mock{}).GradeCheckpoint(context.Background(), GradeInput{LessonInput: in, Items: []tutor.CheckItem{q}, Answers: map[string]string{q.ID: tt.answer}})
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Items []struct {
				Score int `json:"score"`
			} `json:"items"`
		}
		if err := json.Unmarshal(r.Raw, &out); err != nil {
			t.Fatal(err)
		}
		if out.Items[0].Score != tt.want {
			t.Errorf("%q: got %d, want %d", tt.answer, out.Items[0].Score, tt.want)
		}
	}
}
