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
	var seenTutor []ai.TutorInput
	var seenEval []ai.EvalInput
	mock := &ai.Mock{}
	inner := &ai.Mock{}
	mock.TutorFunc = func(in ai.TutorInput) (json.RawMessage, error) {
		seenTutor = append(seenTutor, in)
		return inner.TutorReply(context.Background(), in)
	}
	mock.EvalFunc = func(in ai.EvalInput) (json.RawMessage, error) {
		seenEval = append(seenEval, in)
		return inner.Evaluate(context.Background(), in)
	}
	s := newService(t, mock)
	ctx := context.Background()

	rm, err := s.SaveProfile(WithLang(ctx, "id"), learnerID, course, ProfileInput{Answers: map[string]string{
		"goal": "tech", "background": "other", "background_text": "Data analyst",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if rm.Enrollment.TargetLevel != "B1" || rm.Summary.TargetNodes == 0 {
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

	// The lesson intro gains the goal's line, in the lesson's language.
	sess, err := s.StartSession(WithLang(ctx, "id"), learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(sess.Messages[0].Content, "Tujuanmu untuk kerja, jadi contoh-contohnya akan pakai situasi kantor.") {
		t.Fatalf("intro should be personalised: %q", sess.Messages[0].Content)
	}

	// The tutor and evaluator get the profile.
	view := driveToAssessment(t, s, sess)
	if len(seenTutor) == 0 {
		t.Fatal("tutor was not called")
	}
	in := seenTutor[0]
	if len(in.LearnerProfile) != 2 || !strings.Contains(in.LearnerProfile[0], "work and job interviews") || !strings.Contains(in.LearnerProfile[1], "practice first") {
		t.Fatalf("tutor profile hints wrong: %v", in.LearnerProfile)
	}
	res, err := s.SubmitAssessment(ctx, learnerID, view.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil {
		t.Fatal(err)
	}
	if len(seenEval) != 1 || !strings.Contains(seenEval[0].GoalHint, "work") {
		t.Fatalf("evaluator should get the goal hint: %+v", seenEval)
	}
	// Same answers, same score as without a profile: mastery rules don't move.
	if !res.Passed || res.Attempt.Score != 100 || res.Attempt.Threshold != 80 {
		t.Fatalf("profile must not change scoring: %+v", res.Attempt)
	}
}

func TestNextNodePrefersGoal(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	master := func(node string) {
		t.Helper()
		sess, err := s.StartSession(ctx, learnerID, course, node)
		if err != nil {
			t.Fatal(err)
		}
		view := driveToAssessment(t, s, sess)
		if res, err := s.SubmitAssessment(ctx, learnerID, view.ID, answers(s, node, true)); err != nil || !res.Passed {
			t.Fatalf("master %s: %v", node, err)
		}
	}
	master("a0-alphabet-sounds")

	rm, _ := s.Roadmap(ctx, learnerID, course)
	if rm.NextNodeID == nil || *rm.NextNodeID != "a0-pronouns" {
		t.Fatalf("without a goal, curriculum order: %v", rm.NextNodeID)
	}
	// Travel tags a1-questions (still locked) but not the available A0 nodes,
	// so the suggestion stays in curriculum order rather than jumping ahead.
	rm, err := s.SaveProfile(ctx, learnerID, course, ProfileInput{Answers: map[string]string{"goal": "travel"}})
	if err != nil {
		t.Fatal(err)
	}
	if rm.NextNodeID == nil || *rm.NextNodeID != "a0-pronouns" {
		t.Fatalf("a locked goal node must not be suggested: %v", *rm.NextNodeID)
	}
	if rm.Course.Progress == nil || rm.Course.Progress.Goal == nil || rm.Course.Progress.Goal.ID != "travel" {
		t.Fatalf("course card should carry the goal: %+v", rm.Course.Progress)
	}
}
