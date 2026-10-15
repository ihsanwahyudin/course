package learning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"fluentpath/api/internal/ai"
)

func TestProfileValidationAndSkip(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()

	p, err := s.GetProfile(WithLang(ctx, "id"), learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	if p.Completed || len(p.Questions) == 0 || p.Questions[0].Title != "Untuk apa kamu belajar bahasa Inggris?" || p.TargetLevel != "B2" {
		t.Fatalf("fresh profile view wrong: %+v", p)
	}

	for _, bad := range []ProfileInput{
		{Answers: map[string]string{"background": "it"}},         // goal is required
		{Answers: map[string]string{"goal": "astronaut"}},        // unknown option
		{Answers: map[string]string{"goal": "career", "x": "y"}}, // unknown question
		{Answers: map[string]string{"goal": "career"}, TargetLevel: "Z9"},
		{Answers: map[string]string{"goal": "career", "background": "other", "background_text": strings.Repeat("a", 81)}},
	} {
		if _, err := s.SaveProfile(ctx, learnerID, course, bad); code(err) != "invalid_profile" {
			t.Errorf("%+v: want invalid_profile, got %v", bad, err)
		}
	}
	if rm, _ := s.Roadmap(ctx, learnerID, course); rm.Enrolled {
		t.Fatal("a rejected profile must not enrol the learner")
	}

	rm, err := s.SaveProfile(ctx, learnerID, course, ProfileInput{Skip: true})
	if err != nil {
		t.Fatal(err)
	}
	if !rm.Enrolled || !rm.Profile.Completed || rm.Profile.Goal != nil || rm.Enrollment.TargetLevel != "B2" {
		t.Fatalf("skip should enrol with defaults and stop offering onboarding: %+v", rm.Profile)
	}
}

func TestProfilePersonalisesRoadmapAndTutor(t *testing.T) {
	var seenPlan []ai.LessonInput
	var seenGrade []ai.GradeInput
	mock := &ai.Mock{}
	inner := &ai.Mock{}
	mock.PlanFunc = func(in ai.LessonInput) (json.RawMessage, error) {
		seenPlan = append(seenPlan, in)
		r, err := inner.LessonPlan(context.Background(), in)
		return r.Raw, err
	}
	mock.GradeFunc = func(in ai.GradeInput) (json.RawMessage, error) {
		seenGrade = append(seenGrade, in)
		r, err := inner.GradeCheckpoint(context.Background(), in)
		return r.Raw, err
	}
	s := newService(t, mock)
	withoutAuthoredPlans(s) // personalised plans are AI-written
	ctx := context.Background()

	rm, err := s.SaveProfile(WithLang(ctx, "id"), learnerID, course, ProfileInput{Answers: map[string]string{
		"goal": "tech", "background": "other", "background_text": "Data analyst",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if rm.Enrollment.TargetLevel != "B2" || rm.Summary.TargetNodes == 0 {
		t.Fatalf("goal should set the target level: %+v", rm.Enrollment)
	}
	if !rm.Profile.Completed || rm.Profile.Goal == nil || rm.Profile.Goal.Label != "Membaca dokumentasi teknis" ||
		rm.Profile.Background == nil || rm.Profile.Background.Label != "Data analyst" {
		t.Fatalf("profile summary wrong: %+v %+v", rm.Profile.Goal, rm.Profile.Background)
	}
	rec := map[string]bool{}
	for _, n := range rm.Nodes {
		if n.Recommended {
			rec[n.ID] = true
		}
	}
	if !rec["a1-simple-present"] || !rec["b2-passive-voice"] || rec["a0-alphabet-sounds"] {
		t.Fatalf("recommended nodes wrong: %v", rec)
	}

	// Explicit target wins over the goal's; saving again replaces the answers.
	rm, err = s.SaveProfile(ctx, learnerID, course, ProfileInput{Answers: map[string]string{"goal": "career", "style": "practice"}, TargetLevel: "C1"})
	if err != nil {
		t.Fatal(err)
	}
	if rm.Enrollment.TargetLevel != "C1" || rm.Profile.Background != nil || rm.Profile.Goal.ID != "career" {
		t.Fatalf("resave wrong: %+v %+v", rm.Enrollment, rm.Profile)
	}
	p, _ := s.GetProfile(ctx, learnerID, course)
	if !p.Completed || p.Answers["goal"] != "career" || p.Answers["style"] != "practice" || p.TargetLevel != "C1" {
		t.Fatalf("stored answers wrong: %+v", p)
	}

	// The lesson plan is written for the learner's profile.
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if len(seenPlan) != 1 {
		t.Fatalf("one plan call expected, got %d", len(seenPlan))
	}
	in := seenPlan[0]
	if len(in.LearnerProfile) != 2 || !strings.Contains(in.LearnerProfile[0], "work and job interviews") || !strings.Contains(in.LearnerProfile[1], "practice first") {
		t.Fatalf("tutor profile hints wrong: %v", in.LearnerProfile)
	}
	sess = finishSteps(t, s, sess)
	// The evaluator gets the profile too.
	// One wrong answer: the grader judges it, the exact matches are fixed.
	ans := answers(s, "a0-alphabet-sounds", true)
	ans["q5"] = "g-r-e-n"
	res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, ans)
	if err != nil {
		t.Fatal(err)
	}
	if len(seenGrade) != 1 || !strings.Contains(seenGrade[0].GoalHint, "work") || len(seenGrade[0].Items) != 5 || len(seenGrade[0].Fixed) != 4 {
		t.Fatalf("the grader should get the goal hint, every item and the 4 fixed scores: %+v", seenGrade)
	}
	if res.Attempt.Threshold != 80 {
		t.Fatalf("profile must not change the pass mark: %+v", res.Attempt)
	}
}

func TestNextNodePrefersGoal(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	master := func(node string) {
		t.Helper()
		sess := startDone(t, s, ctx, node)
		if res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, node, true)); err != nil || !res.Passed {
			t.Fatalf("master %s: %v", node, err)
		}
	}
	master("a0-alphabet-sounds")

	rm, _ := s.Roadmap(ctx, learnerID, course)
	if rm.NextNodeID == nil || *rm.NextNodeID != "a0-greetings" {
		t.Fatalf("without a goal, curriculum order: %v", rm.NextNodeID)
	}
	// Travel tags a1-questions (still locked) and a0-greetings, which already
	// leads the curriculum order, so the suggestion does not jump ahead.
	rm, err := s.SaveProfile(ctx, learnerID, course, ProfileInput{Answers: map[string]string{"goal": "travel"}})
	if err != nil {
		t.Fatal(err)
	}
	if rm.NextNodeID == nil || *rm.NextNodeID != "a0-greetings" {
		t.Fatalf("a locked goal node must not be suggested: %v", *rm.NextNodeID)
	}
	if rm.Course.Progress == nil || rm.Course.Progress.Goal == nil || rm.Course.Progress.Goal.ID != "travel" {
		t.Fatalf("course card should carry the goal: %+v", rm.Course.Progress)
	}
}
