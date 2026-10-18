package ai

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/tutor"
)

func TestTutorCallsUseTheVisibleOptionLabels(t *testing.T) {
	in, c := lessonInput(t, "a0-sounds-indonesian", "id")
	p, _, _ := tutor.AuthoredPlan(c, in.Node.ID, "id")
	q := *p.Steps[3].Question
	for name, payload := range map[string]any{"question": publicQuestion(q), "step": publicStep(p.Steps[3])} {
		raw, _ := json.Marshal(payload)
		if !strings.Contains(string(raw), q.DisplayOptions[1]) || strings.Contains(string(raw), q.Options[1]) || strings.Contains(string(raw), q.Explanation) {
			t.Fatalf("%s lost labels or leaked explanation: %s", name, raw)
		}
	}
	if answerText(q) != q.DisplayOptions[q.Answer] {
		t.Fatal("revealed answer must use visible labels")
	}
	similar := similarTask(SimilarInput{LessonInput: in, Question: q})
	if !strings.Contains(similar, `"options":["Panjangkan bunyi i:`) || !strings.Contains(similar, "Explanatory options") {
		t.Fatalf("similar task missing localization: %s", similar)
	}
	// The offline rotation must move each label with its canonical choice.
	in.Lesson = &curriculum.Lesson{}
	res, err := (&Mock{}).SimilarQuestion(context.Background(), SimilarInput{LessonInput: in, Question: q})
	if err != nil {
		t.Fatal(err)
	}
	var rotated tutor.Question
	if err = json.Unmarshal(res.Raw, &rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.Options[rotated.Answer] != q.Options[q.Answer] || answerText(rotated) != answerText(q) {
		t.Fatal("rotation separated label from answer")
	}
}

func TestCheckpointExamplesTranslateExplanatoryOptionsOnly(t *testing.T) {
	in, _ := lessonInput(t, "a1-pron-word-stress", "id")
	it := in.Lesson.Assessment[4]
	ex := exampleOf(it, in.Lesson.PromptIn(it, "id"), in.Lesson.AssessmentLabelsIn(it, "id"))
	if ex.CorrectOption != "kedua" || !slices.Equal(ex.Options, []string{"kedua", "pertama", "ketiga"}) || it.CorrectOption != "second" {
		t.Fatalf("wrong translated sample: %+v", ex)
	}
	task := checkpointTask(CheckpointInput{LessonInput: in, Examples: in.Lesson.Assessment})
	if !strings.Contains(task, `"options":["kedua","pertama","ketiga"]`) || !strings.Contains(task, "Explanatory options follow") {
		t.Fatal("checkpoint writer missing translated sample or language rule")
	}
	items := AuthoredCheckpoint(in.Lesson, "id")
	for _, task := range []string{gradeTask(GradeInput{LessonInput: in, Items: items, Answers: map[string]string{}}), verifyTask(VerifyInput{LessonInput: in, Items: items})} {
		if !strings.Contains(task, `": "kedua"`) || strings.Contains(task, `"second"`) {
			t.Fatal("grader or verifier sees canonical English instead of visible labels")
		}
	}
}
