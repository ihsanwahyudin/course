package learning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/progress"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

// firstQuestion returns the index of the plan's first question step.
func firstQuestion(t *testing.T, p tutor.Plan) int {
	t.Helper()
	for i, st := range p.Steps {
		if st.Question != nil {
			return i
		}
	}
	t.Fatal("plan has no question")
	return -1
}

// stepStates reads the server-side state of every step.
func stepStates(t *testing.T, s *Service, sessionID string) []stepState {
	t.Helper()
	sess, err := storage.GetSession(context.Background(), s.Store.Pool, sessionID, learnerID)
	if err != nil {
		t.Fatal(err)
	}
	var out []stepState
	if err := json.Unmarshal(sess.StepState, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// openStep moves forward until step n is the current one.
func openStep(t *testing.T, s *Service, sess SessionView, n int) SessionView {
	t.Helper()
	for sess.Lesson.StepIndex < n {
		var err error
		if sess, err = s.NextStep(context.Background(), learnerID, sess.ID, sess.Lesson.StepIndex); err != nil {
			t.Fatalf("next %d: %v", sess.Lesson.StepIndex, err)
		}
	}
	return sess
}

func TestStepStateMachine(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := context.Background()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	plan := planOf(t, s, sess.ID)
	qi := firstQuestion(t, plan)
	q := *plan.Steps[qi].Question

	if _, err := s.NextStep(ctx, learnerID, sess.ID, qi+1); code(err) != "step_not_open" {
		t.Fatalf("later steps are closed: %v", err)
	}
	sess = openStep(t, s, sess, qi)
	if len(sess.Lesson.Steps) != qi+1 || !sess.Lesson.Steps[0].Done {
		t.Fatalf("only steps up to the current one are shown: %d", len(sess.Lesson.Steps))
	}
	qv := sess.Lesson.Steps[qi].Question
	if qv == nil || qv.Answer != nil || qv.Solution != "" || qv.Explanation != "" || qv.HintsLeft != tutor.MaxHints {
		t.Fatalf("the answer must stay on the server until answered: %+v", qv)
	}
	raw, _ := json.Marshal(sess)
	if bytes.Contains(raw, []byte(q.Explanation)) {
		t.Fatal("the explanation leaked into the session JSON")
	}

	for want, call := range map[string]func() error{
		"answer_first": func() error { _, err := s.NextStep(ctx, learnerID, sess.ID, qi); return err },
		"hints_first":  func() error { _, err := s.RevealStep(ctx, learnerID, sess.ID, qi, false); return err },
		"not_wrong":    func() error { _, err := s.SimilarStep(ctx, learnerID, sess.ID, qi); return err },
	} {
		if err := call(); code(err) != want {
			t.Errorf("want %s, got %v", want, err)
		}
	}

	// Two hints, then no more.
	for i := range tutor.MaxHints {
		if sess, err = s.HintStep(ctx, learnerID, sess.ID, qi, false); err != nil {
			t.Fatal(err)
		}
		if qv := sess.Lesson.Steps[qi].Question; len(qv.Hints) != i+1 || qv.HintsLeft != tutor.MaxHints-i-1 {
			t.Fatalf("hint %d: %+v", i, qv)
		}
	}
	if _, err := s.HintStep(ctx, learnerID, sess.ID, qi, false); code(err) != "no_hints_left" {
		t.Fatalf("a third hint: %v", err)
	}
	if mock.Calls.Hint != tutor.MaxHints {
		t.Fatalf("hint calls = %d", mock.Calls.Hint)
	}

	// A wrong answer shows the right one and the explanation.
	if sess, err = s.AnswerStep(ctx, learnerID, sess.ID, qi, wrongAnswer(q)); err != nil {
		t.Fatal(err)
	}
	qv = sess.Lesson.Steps[qi].Question
	if !qv.Answered || qv.Correct || qv.Explanation == "" || (q.Type == tutor.QuestionMCQ && (qv.Answer == nil || *qv.Answer != q.Answer)) {
		t.Fatalf("wrong answer result: %+v", qv)
	}
	if _, err := s.AnswerStep(ctx, learnerID, sess.ID, qi, rightAnswer(q)); code(err) != "already_answered" {
		t.Fatalf("answering twice: %v", err)
	}
	if _, err := s.RevealStep(ctx, learnerID, sess.ID, qi, false); code(err) != "already_answered" {
		t.Fatalf("reveal after answering: %v", err)
	}
	bad := -1
	if _, err := s.AnswerStep(ctx, learnerID, sess.ID, qi, AnswerInput{Choice: &bad, Similar: true}); code(err) != "no_similar" {
		t.Fatalf("no similar question yet: %v", err)
	}

	// One similar question (step 3b), then no more.
	if sess, err = s.SimilarStep(ctx, learnerID, sess.ID, qi); err != nil {
		t.Fatal(err)
	}
	sv := sess.Lesson.Steps[qi].Similar
	if sv == nil || sv.Answered || sv.Answer != nil || sv.Prompt == q.Prompt || sess.Lesson.TotalSteps != len(plan.Steps) {
		t.Fatalf("similar question: %+v", sv)
	}
	if _, err := s.SimilarStep(ctx, learnerID, sess.ID, qi); code(err) != "similar_used" {
		t.Fatalf("second similar: %v", err)
	}
	// Get it wrong as well; then the only way is on.
	sim := stepStates(t, s, sess.ID)[qi].Similar.Question
	if sess, err = s.AnswerStep(ctx, learnerID, sess.ID, qi, func() AnswerInput { a := wrongAnswer(sim); a.Similar = true; return a }()); err != nil {
		t.Fatal(err)
	}
	if sv := sess.Lesson.Steps[qi].Similar; !sv.Answered || sv.Correct || sv.Explanation == "" {
		t.Fatalf("similar result: %+v", sv)
	}
	if sess, err = s.NextStep(ctx, learnerID, sess.ID, qi); err != nil || sess.Lesson.StepIndex != qi+1 {
		t.Fatalf("next after the question: %v", err)
	}
	if _, err := s.NextStep(ctx, learnerID, sess.ID, qi); code(err) != "step_done" {
		t.Fatalf("moving past a step twice: %v", err)
	}

	// Finish and pass: the summary lists what to review and the mistakes become flashcards.
	sess = finishSteps(t, s, sess)
	res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil || !res.NewlyMastered {
		t.Fatalf("checkpoint: %+v %v", res, err)
	}
	sum := res.Session.Lesson.Summary
	if sum == nil || len(sum.Mastered) == 0 || sum.FlashcardsAdded != 2 || sum.NextOpening == "" || sum.Next == nil || sum.Next.ID != "a0-sounds-indonesian" {
		t.Fatalf("lesson summary: %+v", sum)
	}
	if mock.Calls.Summary != 1 {
		t.Fatalf("summary calls = %d", mock.Calls.Summary)
	}
	due, err := s.DueFlashcards(ctx, learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	mistakeCards := 0
	for _, c := range due.Cards {
		if c.Source == "mistake" {
			mistakeCards++
		}
	}
	if mistakeCards != 2 {
		t.Fatalf("mistake flashcards = %d", mistakeCards)
	}
	if _, err := s.NextStep(ctx, learnerID, sess.ID, 0); code(err) != "lesson_closed" {
		t.Fatalf("a mastered lesson is closed: %v", err)
	}

	// The next lesson opens with where the previous one left off.
	next, err := s.StartSession(ctx, learnerID, course, "a0-pronouns")
	if err != nil {
		t.Fatal(err)
	}
	if chip := next.Lesson.MemoryChip; chip == nil || chip.From != "Alphabet & Sounds" || chip.Strong == "" {
		t.Fatalf("memory chip: %+v", next.Lesson.MemoryChip)
	}
}

func TestRevealAfterTwoHints(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	qi := firstQuestion(t, planOf(t, s, sess.ID))
	sess = openStep(t, s, sess, qi)
	for range tutor.MaxHints {
		sess, _ = s.HintStep(ctx, learnerID, sess.ID, qi, false)
	}
	sess, err := s.RevealStep(ctx, learnerID, sess.ID, qi, false)
	if err != nil {
		t.Fatal(err)
	}
	if qv := sess.Lesson.Steps[qi].Question; !qv.Revealed || qv.Answered || qv.Explanation == "" {
		t.Fatalf("revealed question: %+v", qv)
	}
	// Giving up counts as a mistake: a similar question is offered.
	if _, err := s.SimilarStep(ctx, learnerID, sess.ID, qi); err != nil {
		t.Fatalf("similar after giving up: %v", err)
	}
}

func TestLeakingHintsAreReplaced(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	plan := planOf(t, s, sess.ID)
	qi := firstQuestion(t, plan)
	q := *plan.Steps[qi].Question
	leak := q.Solution
	if q.Type == tutor.QuestionMCQ {
		leak = q.Options[q.Answer]
	}
	mock.HintFunc = func(ai.HintInput) (json.RawMessage, error) {
		return json.Marshal(tutor.Hint{Hint: "The answer is " + leak})
	}
	sess = openStep(t, s, sess, qi)
	sess, err := s.HintStep(ctx, learnerID, sess.ID, qi, false)
	if err != nil {
		t.Fatal(err)
	}
	if h := sess.Lesson.Steps[qi].Question.Hints[0]; h != fallbackHints["en"][0] {
		t.Fatalf("a leaking hint must be replaced: %q", h)
	}
	if mock.Calls.Hint != 2 {
		t.Fatalf("a rejected hint is retried once: %d calls", mock.Calls.Hint)
	}
}

func TestAskScopeAndMemory(t *testing.T) {
	var seen []ai.AskInput
	reply := tutor.Answer{Answer: "We cover past tense later.", Scope: tutor.ScopeOtherLesson, OtherLesson: "Past Simple", Remember: "Works as a nurse"}
	mock := &ai.Mock{AskFunc: func(in ai.AskInput) (json.RawMessage, error) {
		seen = append(seen, in)
		return json.Marshal(reply)
	}}
	s := newService(t, mock)
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")

	if _, err := s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: "  "}); code(err) != "empty_message" {
		t.Fatalf("empty: %v", err)
	}
	if _, err := s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: strings.Repeat("a", MaxMessageRunes+1)}); code(err) != "message_too_long" {
		t.Fatalf("long: %v", err)
	}
	if _, err := s.AskStep(ctx, learnerID, sess.ID, 1, AskInput{Content: "hi"}); code(err) != "step_not_open" {
		t.Fatalf("asking in a closed step: %v", err)
	}
	sess, err := s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: "How do I say 'kemarin saya bekerja'?"})
	if err != nil {
		t.Fatal(err)
	}
	a := sess.Lesson.Steps[0].Asks
	if len(a) != 1 || a[0].Scope != tutor.ScopeOtherLesson || a[0].OtherLesson != "Past Simple" {
		t.Fatalf("asks: %+v", a)
	}
	in := seen[0]
	if in.Step.Title != sess.Lesson.Steps[0].Title || len(in.OtherLessons) == 0 || strings.Contains(strings.Join(in.OtherLessons, "|"), "Alphabet & Sounds") {
		t.Fatalf("ask input: step %q, other lessons %v", in.Step.Title, in.OtherLessons)
	}
	mem, _ := s.Memory(ctx, learnerID)
	if !mem.Enabled || len(mem.Items) != 1 || mem.Items[0].Text != "Works as a nurse" || mem.Items[0].Kind != "personal" {
		t.Fatalf("memory: %+v", mem)
	}

	// The next question carries the step's history and the memory.
	reply.Remember = "Likes football"
	if _, err := s.SetMemoryEnabled(ctx, learnerID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: "And tomorrow?"}); err != nil {
		t.Fatal(err)
	}
	if in := seen[1]; len(in.History) != 2 || in.History[0].Content != "How do I say 'kemarin saya bekerja'?" || len(in.Memory) != 1 {
		t.Fatalf("second ask input: history %+v memory %v", in.History, in.Memory)
	}
	if mem, _ := s.Memory(ctx, learnerID); mem.Enabled || len(mem.Items) != 1 {
		t.Fatalf("with memory off nothing new is saved: %+v", mem)
	}

	// Moving forward and chatting elsewhere must preserve this step's thread.
	if _, err := s.NextStep(ctx, learnerID, sess.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AskStep(ctx, learnerID, sess.ID, 1, AskInput{Content: "Question on the next step"}); err != nil {
		t.Fatal(err)
	}
	sess, err = s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: "Back to my earlier question"})
	if err != nil {
		t.Fatal(err)
	}
	if in := seen[len(seen)-1]; len(in.History) != 4 || in.History[0].Content != "How do I say 'kemarin saya bekerja'?" || in.History[2].Content != "And tomorrow?" {
		t.Fatalf("reopened step lost or mixed its context: %+v", in.History)
	}
	sess, err = s.GetSession(ctx, learnerID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Lesson.StepIndex != 1 || len(sess.Lesson.Steps[0].Asks) != 3 || len(sess.Lesson.Steps[1].Asks) != 1 {
		t.Fatalf("chatting on an earlier step changed progress or history: %+v", sess.Lesson)
	}
}

// withoutAuthoredPlans makes the AI write plans, as with personalisation on.
func withoutAuthoredPlans(s *Service) {
	cur, _ := s.Catalog.Course(course)
	cur.DropAuthoredPlans()
}

func TestHandWrittenStepsNeedNoAI(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := context.Background()
	for _, lang := range []string{"en", "id"} {
		other := newUser(t, s, lang+"@example.com")
		sess, err := s.StartSession(WithLang(ctx, lang), other, course, "a0-alphabet-sounds")
		if err != nil {
			t.Fatal(err)
		}
		cur, _ := s.Catalog.Course(course)
		want, ok, err := tutor.AuthoredPlan(cur, "a0-alphabet-sounds", lang)
		if !ok || err != nil {
			t.Fatalf("%s: authored plan: %v %v", lang, ok, err)
		}
		if sess.Lesson.TotalSteps != len(want.Steps) || sess.Lesson.Steps[0].Title != want.Steps[0].Title || sess.Lesson.Greeting != want.Greeting {
			t.Fatalf("%s: the lesson should use its hand-written steps: %+v", lang, sess.Lesson.Steps[0])
		}
	}
	if mock.Calls.Plan != 0 {
		t.Fatalf("hand-written lessons must not call the AI for a plan: %d calls", mock.Calls.Plan)
	}
}

func TestLessonFollowsLanguageSwitch(t *testing.T) {
	s := newService(t, &ai.Mock{})
	en, id := WithLang(context.Background(), "en"), WithLang(context.Background(), "id")
	cur, _ := s.Catalog.Course(course)
	enPlan, _, _ := tutor.AuthoredPlan(cur, "a0-alphabet-sounds", "en")
	idPlan, _, _ := tutor.AuthoredPlan(cur, "a0-alphabet-sounds", "id")

	sess, err := s.StartSession(en, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if sess, err = s.NextStep(en, learnerID, sess.ID, 0); err != nil {
		t.Fatal(err)
	}
	q := enPlan.Steps[1].Question
	if q == nil {
		t.Fatal("step 2 of the alphabet lesson should be a question")
	}
	if sess, err = s.AnswerStep(en, learnerID, sess.ID, 1, rightAnswer(*q)); err != nil {
		t.Fatal(err)
	}

	// Switching to Indonesian: same place, same answer, Indonesian steps.
	got, err := s.GetSession(id, learnerID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Locale != "id" || got.Lesson.StepIndex != 1 || got.Lesson.Steps[0].Title != idPlan.Steps[0].Title || got.Lesson.Greeting != idPlan.Greeting {
		t.Fatalf("the lesson should switch to Indonesian in place: %s step %d %q", got.Locale, got.Lesson.StepIndex, got.Lesson.Steps[0].Title)
	}
	if a := got.Lesson.Steps[1].Question; a == nil || !a.Answered || !a.Correct || a.Prompt != idPlan.Steps[1].Question.Prompt {
		t.Fatalf("the answered question should carry over: %+v", a)
	}
	// And back, also through StartSession.
	if back, err := s.StartSession(en, learnerID, course, "a0-alphabet-sounds"); err != nil || back.ID != sess.ID || back.Lesson.Steps[0].Title != enPlan.Steps[0].Title {
		t.Fatalf("back to English: %v %s %q", err, back.ID, back.Lesson.Steps[0].Title)
	}
}

func TestCorrectedWordingKeepsTheLearnersPlace(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if sess, err = s.NextStep(ctx, learnerID, sess.ID, 0); err != nil {
		t.Fatal(err)
	}
	cur, _ := s.Catalog.Course(course)
	current, _, _ := tutor.AuthoredPlan(cur, "a0-alphabet-sounds", "en")
	setStored := func(p tutor.Plan) {
		raw, _ := json.Marshal(p)
		if err := storage.SetSessionLanguage(ctx, s.Store.Pool, sess.ID, "en", raw); err != nil {
			t.Fatal(err)
		}
	}

	// An older wording of the same steps: updated in place, same step.
	old := current
	old.Steps = append([]tutor.Step{}, current.Steps...)
	old.Steps[0].Body = "An older explanation."
	setStored(old)
	got, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil || got.ID != sess.ID || got.Lesson.StepIndex != 1 || got.Lesson.Steps[0].Body != current.Steps[0].Body {
		t.Fatalf("new wording should apply in place: %v %s step %d %q", err, got.ID, got.Lesson.StepIndex, got.Lesson.Steps[0].Body)
	}

	// Different questions: a new version of the lesson.
	q := *current.Steps[1].Question
	q.Options = append([]string{"Z"}, q.Options[1:]...)
	old.Steps[1].Question = &q
	setStored(old)
	if got, err = s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds"); err != nil || got.ID == sess.ID || got.Lesson.StepIndex != 0 {
		t.Fatalf("changed questions start the new version: %v %s", err, got.ID)
	}
}

func TestPersonalisationOff(t *testing.T) {
	s := newService(t, &ai.Mock{})
	s.Catalog.DisableOnboarding()
	ctx := context.Background()
	rm, err := s.Roadmap(ctx, learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	if rm.Profile.Available {
		t.Fatal("onboarding should be off")
	}
	if p, err := s.GetProfile(ctx, learnerID, course); err != nil || len(p.Questions) != 0 {
		t.Fatalf("no onboarding questions: %+v %v", p.Questions, err)
	}
	if rm, err = s.Enroll(ctx, learnerID, course); err != nil || !rm.Enrolled || rm.Enrollment.TargetLevel != "B2" {
		t.Fatalf("enrolling works without onboarding: %+v %v", rm.Enrollment, err)
	}
	for _, n := range rm.Nodes {
		if n.Recommended {
			t.Fatalf("no goal badges while personalisation is off: %s", n.ID)
		}
	}
	if mem, _ := s.Memory(ctx, learnerID); len(mem.Profile) != 0 {
		t.Fatalf("no onboarding answers in memory: %+v", mem.Profile)
	}
}

func TestPlanFallbackAndCache(t *testing.T) {
	mock := &ai.Mock{PlanFunc: func(ai.LessonInput) (json.RawMessage, error) {
		return json.RawMessage(`{"greeting":"hi","steps":[]}`), nil
	}}
	s := newService(t, mock)
	withoutAuthoredPlans(s)
	ctx := context.Background()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if mock.Calls.Plan != 2 || sess.Lesson == nil || sess.Lesson.TotalSteps < tutor.MinSteps {
		t.Fatalf("an invalid plan is retried once, then the material's plan is used: %d calls, %+v", mock.Calls.Plan, sess.Lesson)
	}

	// A valid plan is cached and shared by learners with the same profile.
	mock = &ai.Mock{}
	s.AI = mock
	other := newUser(t, s, "other@example.com")
	third := newUser(t, s, "third@example.com")
	if _, err := s.StartSession(ctx, other, course, "a0-alphabet-sounds"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartSession(ctx, third, course, "a0-alphabet-sounds"); err != nil {
		t.Fatal(err)
	}
	if mock.Calls.Plan != 1 {
		t.Fatalf("plan calls = %d, want 1 (cached)", mock.Calls.Plan)
	}
	// Resuming a session keeps its plan.
	if _, err := s.StartSession(ctx, other, course, "a0-alphabet-sounds"); err != nil || mock.Calls.Plan != 1 {
		t.Fatalf("resume: %v, %d calls", err, mock.Calls.Plan)
	}
}

func TestMemoryCRUD(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	for _, txt := range []string{"Works as a nurse", "Often mixes he and she", "Works as a nurse"} {
		if err := storage.AddMemory(ctx, s.Store.Pool, learnerID, course, "personal", txt, "a0-pronouns"); err != nil {
			t.Fatal(err)
		}
	}
	mem, err := s.Memory(ctx, learnerID)
	if err != nil || len(mem.Items) != 2 {
		t.Fatalf("duplicates are skipped: %+v %v", mem, err)
	}
	id := mem.Items[0].ID
	if _, err := s.UpdateMemory(ctx, learnerID, id, " "); code(err) != "invalid_memory" {
		t.Fatalf("empty: %v", err)
	}
	if _, err := s.UpdateMemory(ctx, learnerID, id, strings.Repeat("x", 201)); code(err) != "invalid_memory" {
		t.Fatalf("long: %v", err)
	}
	other := newUser(t, s, "other@example.com")
	if _, err := s.UpdateMemory(ctx, other, id, "hacked"); code(err) != "memory_not_found" {
		t.Fatalf("other user edited a memory: %v", err)
	}
	if _, err := s.DeleteMemory(ctx, other, id); code(err) != "memory_not_found" {
		t.Fatalf("other user deleted a memory: %v", err)
	}
	if om, _ := s.Memory(ctx, other); len(om.Items) != 0 {
		t.Fatalf("other user sees memory: %+v", om)
	}
	if mem, err = s.UpdateMemory(ctx, learnerID, id, "Works as a  night nurse"); err != nil || mem.Items[0].Text != "Works as a night nurse" {
		t.Fatalf("update: %+v %v", mem.Items, err)
	}
	if mem, err = s.DeleteMemory(ctx, learnerID, id); err != nil || len(mem.Items) != 1 {
		t.Fatalf("delete: %+v %v", mem.Items, err)
	}
	if mem, err = s.DeleteAllMemory(ctx, learnerID); err != nil || len(mem.Items) != 0 {
		t.Fatalf("delete all: %+v %v", mem.Items, err)
	}

	// Onboarding answers are listed (read-only) with the memory.
	if _, err := s.SaveProfile(ctx, learnerID, course, ProfileInput{Answers: map[string]string{"goal": "career"}}); err != nil {
		t.Fatal(err)
	}
	if mem, _ = s.Memory(ctx, learnerID); len(mem.Profile) != 1 || mem.Profile[0].Course != course {
		t.Fatalf("profile facts: %+v", mem.Profile)
	}
}

func TestNotes(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	art := tutor.Artifact{Title: "Cheat sheet · Pronouns", Headers: []string{"Topic", "Meaning"}, Rows: [][]string{{"he", "dia (lk)"}}}
	if _, err := s.SaveNote(ctx, learnerID, NoteInput{Course: course, NodeID: "a0-pronouns", Artifact: tutor.Artifact{Title: " "}}); code(err) != "invalid_note" {
		t.Fatalf("untitled: %v", err)
	}
	if _, err := s.SaveNote(ctx, learnerID, NoteInput{Course: course, NodeID: "nope", Artifact: art}); code(err) != "node_not_found" {
		t.Fatalf("unknown lesson: %v", err)
	}
	big := art
	for i := range 31 {
		big.Rows = append(big.Rows, []string{fmt.Sprint(i), "x"})
	}
	if _, err := s.SaveNote(ctx, learnerID, NoteInput{Course: course, NodeID: "a0-pronouns", Artifact: big}); code(err) != "invalid_note" {
		t.Fatalf("too many rows: %v", err)
	}
	n, err := s.SaveNote(ctx, learnerID, NoteInput{Course: course, NodeID: "a0-pronouns", Artifact: art})
	if err != nil || n.NodeTitle != "Personal Pronouns" || n.CourseTitle != "English" {
		t.Fatalf("save: %+v %v", n, err)
	}
	// Saving the same sheet again replaces it.
	art.Rows = append(art.Rows, []string{"she", "dia (pr)"})
	if _, err := s.SaveNote(ctx, learnerID, NoteInput{Course: course, NodeID: "a0-pronouns", Artifact: art}); err != nil {
		t.Fatal(err)
	}
	notes, _ := s.Notes(ctx, learnerID)
	if len(notes) != 1 || !strings.Contains(string(notes[0].Content), "dia (pr)") {
		t.Fatalf("notes: %+v", notes)
	}
	other := newUser(t, s, "other@example.com")
	if on, _ := s.Notes(ctx, other); len(on) != 0 {
		t.Fatalf("other user sees notes: %+v", on)
	}
	if err := s.DeleteNote(ctx, other, notes[0].ID); code(err) != "note_not_found" {
		t.Fatalf("other user deleted a note: %v", err)
	}
	if err := s.DeleteNote(ctx, learnerID, notes[0].ID); err != nil {
		t.Fatal(err)
	}
}

// audioMock is the offline provider plus fake speech. delay, when set,
// makes each speech call wait (like a slow provider).
type audioMock struct {
	*ai.Mock
	delay              time.Duration
	speech, transcribe atomic.Int32
}

func (m *audioMock) Speech(ctx context.Context, text string) ([]byte, error) {
	m.speech.Add(1)
	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return []byte("mp3:" + text), nil
}

func (m *audioMock) Transcribe(ctx context.Context, audio []byte, filename string) (string, error) {
	m.transcribe.Add(1)
	return "I am a nurse.", nil
}

func TestSpeechAndTranscription(t *testing.T) {
	s := newService(t, &ai.Mock{})
	s.LiveSpeech = true
	s.SpeechCacheDir = t.TempDir()
	ctx := context.Background()
	for _, txt := range []string{" ", strings.Repeat("a", maxSpeechRunes+1)} {
		if _, err := s.Speech(ctx, learnerID, txt); code(err) != "invalid_text" {
			t.Fatalf("speech %d chars: %v", len(txt), err)
		}
	}
	if _, err := s.Speech(ctx, learnerID, "Hello"); code(err) != "speech_unavailable" {
		t.Fatalf("the offline provider has no voice: %v", err)
	}
	for _, audio := range [][]byte{nil, make([]byte, MaxAudioBytes+1)} {
		if _, err := s.Transcribe(ctx, learnerID, audio, "a.m4a"); code(err) != "invalid_audio" {
			t.Fatalf("audio of %d bytes: %v", len(audio), err)
		}
	}
	if _, err := s.Transcribe(ctx, learnerID, []byte("x"), "a.m4a"); code(err) != "speech_unavailable" {
		t.Fatalf("offline transcription: %v", err)
	}

	am := &audioMock{Mock: &ai.Mock{}}
	s.AI = am
	for range 2 {
		b, err := s.Speech(ctx, learnerID, "He is  busy.")
		if err != nil || string(b) != "mp3:He is busy." {
			t.Fatalf("speech: %q %v", b, err)
		}
	}
	if am.speech.Load() != 1 {
		t.Fatalf("speech is cached: %d calls", am.speech.Load())
	}
	if txt, err := s.Transcribe(ctx, learnerID, []byte("audio"), "a.m4a"); err != nil || txt != "I am a nurse." {
		t.Fatalf("transcribe: %q %v", txt, err)
	}
}

func TestSlowSpeechIsSharedAndStillCached(t *testing.T) {
	s := newService(t, &ai.Mock{})
	s.LiveSpeech = true
	s.SpeechCacheDir = t.TempDir()
	am := &audioMock{Mock: &ai.Mock{}, delay: 300 * time.Millisecond}
	s.AI = am

	// Three requests for the same sentence at once make one provider call.
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b, err := s.Speech(context.Background(), learnerID, "They work remotely."); err != nil || string(b) != "mp3:They work remotely." {
				t.Errorf("speech: %q %v", b, err)
			}
		}()
	}
	wg.Wait()
	if n := am.speech.Load(); n != 1 {
		t.Fatalf("concurrent requests should share one call, got %d", n)
	}

	// A requester that gives up doesn't waste the call: it finishes and is cached.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := s.Speech(ctx, learnerID, "Her report is on the drive."); code(err) != "speech_failed" {
		t.Fatalf("a requester that left: %v", err)
	}
	time.Sleep(400 * time.Millisecond)
	start := time.Now()
	if b, err := s.Speech(context.Background(), learnerID, "Her report is on the drive."); err != nil || string(b) != "mp3:Her report is on the drive." || time.Since(start) > 100*time.Millisecond {
		t.Fatalf("the finished audio should come from the cache: %q %v %s", b, err, time.Since(start))
	}
	if n := am.speech.Load(); n != 2 {
		t.Fatalf("provider calls = %d, want 2", n)
	}
}

// withoutRecordedAudio hides the course's committed lesson audio.
func withoutRecordedAudio(t *testing.T, s *Service) {
	cur, _ := s.Catalog.Course(course)
	cur.Dir = t.TempDir()
}

func TestLessonAudioIsMadeAhead(t *testing.T) {
	s := newService(t, &ai.Mock{})
	withoutRecordedAudio(t, s)
	s.LiveSpeech = true
	s.SpeechCacheDir = t.TempDir()
	am := &audioMock{Mock: &ai.Mock{}}
	s.AI = am
	s.WarmSpeech = true
	ctx := context.Background()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	s.warm.Wait()
	texts := tutor.SpeechTexts(planOf(t, s, sess.ID), warmMaxTexts)
	if len(texts) == 0 || int(am.speech.Load()) != len(texts) {
		t.Fatalf("every word and example should be made ahead: %d texts, %d calls", len(texts), am.speech.Load())
	}
	before := am.speech.Load()
	if _, err := s.Speech(ctx, learnerID, texts[0]); err != nil || am.speech.Load() != before {
		t.Fatalf("listening later should hit the cache: %v, %d calls", err, am.speech.Load())
	}
	// Resuming the lesson doesn't make it again.
	if _, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds"); err != nil {
		t.Fatal(err)
	}
	s.warm.Wait()
	if am.speech.Load() != before {
		t.Fatalf("resume re-made audio: %d calls", am.speech.Load())
	}
}

// recordedLesson is an open lesson whose audio isn't waiting for review
// (audio/pending.json), so startup warms it like every other.
func recordedLesson(t *testing.T, s *Service) string {
	t.Helper()
	cur, _ := s.Catalog.Course(course)
	rm, err := s.Roadmap(context.Background(), learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range rm.Nodes {
		if _, ok := cur.Lesson(n.ID); ok && n.Status == progress.StatusAvailable && !cur.AudioPending(n.ID) {
			return n.ID
		}
	}
	t.Fatal("no open lesson with recorded audio")
	return ""
}

func TestMissingLessonAudioIsMadeAtStartup(t *testing.T) {
	s := newService(t, &ai.Mock{})
	withoutRecordedAudio(t, s)
	s.LiveSpeech = true
	s.SpeechCacheDir = t.TempDir()
	am := &audioMock{Mock: &ai.Mock{}}
	s.AI = am
	s.WarmSpeech = true
	s.WarmAllLessons()
	s.warm.Wait()
	made := am.speech.Load()
	if made < 100 {
		t.Fatalf("every lesson's sentences should be made: %d", made)
	}
	// Already cached: a restart makes nothing new, and opening a lesson plays from the cache.
	s.WarmAllLessons()
	s.warm.Wait()
	if _, err := s.StartSession(context.Background(), learnerID, course, recordedLesson(t, s)); err != nil {
		t.Fatal(err)
	}
	s.warm.Wait()
	if am.speech.Load() != made {
		t.Fatalf("cached audio was made again: %d → %d", made, am.speech.Load())
	}
	// Shared lesson audio isn't charged to a learner.
	if u, _ := storage.UsageOn(context.Background(), s.Store.Pool, learnerID, s.day()); u.Output != 0 {
		t.Fatalf("learner charged for shared audio: %+v", u)
	}
}

func TestRecordedLessonAudioNeedsNoProvider(t *testing.T) {
	s := newService(t, &ai.Mock{})
	s.LiveSpeech = true
	s.SpeechCacheDir = t.TempDir()
	am := &audioMock{Mock: &ai.Mock{}}
	s.AI = am
	s.WarmSpeech = true
	ctx := context.Background()
	s.WarmAllLessons()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	s.warm.Wait()
	text := sess.Lesson.Steps[0].Pairs[0].EN
	cur, _ := s.Catalog.Course(course)
	want, _ := os.ReadFile(cur.AudioPath(text))
	b, err := s.Speech(ctx, learnerID, text)
	if err != nil || len(b) == 0 || !bytes.Equal(b, want) {
		t.Fatalf("lesson audio should come from the repo: %d bytes, %v", len(b), err)
	}
	if n := am.speech.Load(); n != 0 {
		t.Fatalf("the provider was called %d times for recorded lesson audio", n)
	}

	// Bundling writes every sentence once; a second run writes nothing.
	cur.Dir = t.TempDir()
	written, failed := s.BundleLessonAudio(ctx, false)
	if len(failed) != 0 || written != len(tutor.LessonSpeechTexts(cur)) {
		t.Fatalf("bundle: %d written, %d failed", written, len(failed))
	}
	if written, _ = s.BundleLessonAudio(ctx, false); written != 0 {
		t.Fatalf("second bundle wrote %d", written)
	}
	// -fresh re-records everything from the provider (new voice or model).
	before := am.speech.Load()
	if written, _ = s.BundleLessonAudio(ctx, true); written != len(tutor.LessonSpeechTexts(cur)) || int(am.speech.Load()-before) != written {
		t.Fatalf("fresh bundle: %d written, %d provider calls", written, am.speech.Load()-before)
	}
}

func TestRewrittenLessonStartsTheNewVersion(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: "Why is A read 'ei'?"}); err != nil {
		t.Fatal(err)
	}
	// Same steps: resuming keeps the session.
	if again, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds"); again.ID != sess.ID {
		t.Fatal("an unchanged lesson should resume")
	}
	// The session holds an older version of the lesson, with other steps.
	old := planOf(t, s, sess.ID)
	old.Steps = old.Steps[:len(old.Steps)-1]
	old.Steps[0].Title = "An older title"
	raw, _ := json.Marshal(old)
	if err := storage.SetSessionPlan(ctx, s.Store.Pool, sess.ID, raw); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil || fresh.ID == sess.ID || fresh.Lesson.Steps[0].Title == "An older title" {
		t.Fatalf("a rewritten lesson should start its new version: %v", err)
	}
	// The old session and its question are kept.
	kept, err := s.GetSession(ctx, learnerID, sess.ID)
	if err != nil || kept.EndedAt == nil {
		t.Fatalf("old session should be closed but kept: %v", err)
	}
}

func TestNoAudioIsMadeUnlessAskedFor(t *testing.T) {
	s := newService(t, &ai.Mock{})
	withoutRecordedAudio(t, s)
	s.SpeechCacheDir = t.TempDir()
	am := &audioMock{Mock: &ai.Mock{}}
	s.AI = am
	s.WarmSpeech = true // even if warming is on, LiveSpeech off makes nothing
	ctx := context.Background()
	s.WarmAllLessons()
	if _, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds"); err != nil {
		t.Fatal(err)
	}
	s.warm.Wait()
	if _, err := s.Speech(ctx, learnerID, "They work remotely."); code(err) != "speech_unavailable" {
		t.Fatalf("missing audio should fall back to the device voice: %v", err)
	}
	if n := am.speech.Load(); n != 0 {
		t.Fatalf("no audio may be made without bundle-audio: %d provider calls", n)
	}
}

func TestRelatedTracksAndFoundations(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := WithLang(context.Background(), "id")
	rm, err := s.Roadmap(ctx, learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	if len(rm.RelatedTracks) != 4 || rm.RelatedTracks[0].Slug != "toefl-itp" || rm.RelatedTracks[0].Status != "coming_soon" {
		t.Fatalf("related tracks: %+v", rm.RelatedTracks)
	}
	if r := rm.RelatedTracks[0].RecommendedAfter; r == nil || r.Level != "B1" || r.CourseTitle != "Bahasa Inggris" || r.LevelTitle == "" {
		t.Fatalf("recommended after: %+v", r)
	}
	// A coming-soon track can be browsed but not started.
	track, err := s.Roadmap(ctx, learnerID, "toefl-itp")
	if err != nil || len(track.Nodes) == 0 {
		t.Fatalf("track roadmap: %v", err)
	}
	if _, err := s.Enroll(ctx, learnerID, "toefl-itp"); code(err) != "course_coming_soon" {
		t.Fatalf("enrolling in a coming-soon track: %v", err)
	}
	// A track lesson lists the English lessons it builds on, with the learner's status there.
	var withFoundations string
	cur, _ := s.Catalog.Course("toefl-itp")
	for _, n := range cur.Nodes {
		if len(n.Foundations) > 0 {
			withFoundations = n.ID
			break
		}
	}
	d, err := s.Node(ctx, learnerID, "toefl-itp", withFoundations)
	if err != nil || len(d.Foundations) == 0 || d.Foundations[0].Course.Slug != "english" || d.Foundations[0].Status == "" {
		t.Fatalf("foundations: %+v %v", d.Foundations, err)
	}
}

func TestListeningQuestions(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	plan := planOf(t, s, sess.ID)
	qi := firstQuestion(t, plan)
	plan.Steps[qi].Question.Listen = "The parcel arrives on Thursday."
	raw, _ := json.Marshal(plan)
	if err := storage.SetSessionPlan(ctx, s.Store.Pool, sess.ID, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StepListen(ctx, learnerID, sess.ID, qi); code(err) != "step_not_open" {
		t.Fatalf("a closed step can't be played: %v", err)
	}
	sess, _ = s.GetSession(ctx, learnerID, sess.ID)
	sess = openStep(t, s, sess, qi)
	qv := sess.Lesson.Steps[qi].Question
	if !qv.HasAudio || qv.Transcript != "" {
		t.Fatalf("listening question before answering: %+v", qv)
	}
	raw, _ = json.Marshal(sess)
	if bytes.Contains(raw, []byte("parcel arrives")) {
		t.Fatal("the listening text leaked into the session")
	}
	// No recording (offline provider): the text comes back for the device's voice.
	a, err := s.StepListen(ctx, learnerID, sess.ID, qi)
	if err != nil || a.Audio != nil || a.Text != "The parcel arrives on Thursday." {
		t.Fatalf("listen fallback: %+v %v", a, err)
	}
	other := newUser(t, s, "other@example.com")
	if _, err := s.StepListen(ctx, other, sess.ID, qi); code(err) != "session_not_found" {
		t.Fatalf("another user played the audio: %v", err)
	}
	sess, err = s.AnswerStep(ctx, learnerID, sess.ID, qi, wrongAnswer(*plan.Steps[qi].Question))
	if err != nil || sess.Lesson.Steps[qi].Question.Transcript == "" {
		t.Fatalf("the transcript shows after answering: %v", err)
	}
	if _, err := s.SimilarStep(ctx, learnerID, sess.ID, qi); code(err) != "similar_not_for_listening" {
		t.Fatalf("listening questions have no similar question: %v", err)
	}
}
