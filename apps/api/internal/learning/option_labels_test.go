package learning

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

func TestTranslatedOptionsUpdateExistingSessionWithoutLosingAnswers(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := WithLang(context.Background(), "id")
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	cur, _ := s.Catalog.Course(course)
	plan, _, _ := tutor.AuthoredPlan(cur, "a0-alphabet-sounds", "id")
	// Reach and answer the question whose choices need translated labels.
	for i := 0; i <= 4; i++ {
		if q := plan.Steps[i].Question; q != nil {
			sess, err = s.AnswerStep(ctx, learnerID, sess.ID, i, rightAnswer(*q))
			if err != nil {
				t.Fatal(err)
			}
		}
		if i < 4 {
			sess, err = s.NextStep(ctx, learnerID, sess.ID, i)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	// Simulate a session stored before translated option labels existed.
	for _, st := range plan.Steps {
		if st.Question != nil {
			st.Question.DisplayOptions = nil
		}
	}
	raw, _ := json.Marshal(plan)
	if err = storage.SetSessionLanguage(ctx, s.Store.Pool, sess.ID, "id", raw); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSession(ctx, learnerID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	q := got.Lesson.Steps[4].Question
	if got.ID != sess.ID || got.Lesson.StepIndex != 4 || !q.Answered || !q.Correct || !slices.Equal(q.Options, []string{"I dan E", "Hanya I", "B dan K"}) || q.Answer == nil || *q.Answer != 0 {
		t.Fatalf("translation must update in place and retain answer: %+v, step %d", q, got.Lesson.StepIndex)
	}
	back, err := s.GetSession(WithLang(ctx, "en"), learnerID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	bq := back.Lesson.Steps[4].Question
	if back.ID != sess.ID || !bq.Answered || !bq.Correct || bq.Options[0] != "I and E" {
		t.Fatalf("switch back lost progress or English: %+v", bq)
	}
}

func TestScreenshotQuestionExposesIndonesianOptions(t *testing.T) {
	cat, err := curriculum.LoadCourse("../../../../packages/courses/english")
	if err != nil {
		t.Fatal(err)
	}
	p, _, err := tutor.AuthoredPlan(cat, "a0-sounds-indonesian", "id")
	if err != nil {
		t.Fatal(err)
	}
	v := questionView(*p.Steps[3].Question, qState{})
	if v.Options[1] != "Letakkan ujung lidah di antara gigi untuk bunyi th" || v.Answer != nil {
		t.Fatalf("wrong visible options or answer exposed: %+v", v)
	}
}
