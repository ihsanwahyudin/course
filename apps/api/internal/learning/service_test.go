package learning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/progress"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

const course = "english"

// learnerID is the user created by newService; tests that need a second user call newUser.
var learnerID string

// newService returns a service backed by a fresh schema in TEST_DATABASE_URL.
func newService(t *testing.T, provider ai.Provider) *Service {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", rand.Uint64())
	admin, err := storage.Open(ctx, url, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	store, err := storage.Open(ctx, url, schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		store.Close()
		_, _ = admin.Pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	cat, err := curriculum.LoadCatalog("../../../../packages/courses")
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{Store: store, Catalog: cat, AI: provider, TimeZone: "Asia/Jakarta"}
	learnerID = newUser(t, s, "learner@example.com")
	return s
}

func newUser(t *testing.T, s *Service, email string) string {
	t.Helper()
	u, err := storage.CreateUser(context.Background(), s.Store.Pool, email, "x", "Test", "en")
	if err != nil {
		t.Fatal(err)
	}
	return u.ID
}

func lessonOf(s *Service, nodeID string) *curriculum.Lesson {
	cur, _ := s.Catalog.Course(course)
	l, _ := cur.Lesson(nodeID)
	return l
}

func code(err error) string {
	var le *Error
	if errors.As(err, &le) {
		return le.Code
	}
	return ""
}

func answers(s *Service, nodeID string, correct bool) map[string]string {
	lesson := lessonOf(s, nodeID)
	out := map[string]string{}
	for _, it := range lesson.Assessment {
		if correct {
			out[it.ID] = it.AcceptableAnswers[0]
		} else {
			out[it.ID] = "zzz"
		}
	}
	return out
}

func TestFullLoopMastersAndUnlocks(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()

	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if sess.Lesson == nil || len(sess.Lesson.Steps) != 1 || sess.Lesson.TotalSteps < tutor.MinSteps || sess.Material == nil || len(sess.Assessment) == 0 || sess.Phase != string(tutor.PhaseLearning) {
		t.Fatalf("a new lesson opens on its first step with the material at hand: %+v", sess)
	}
	if sess.Prev == nil || sess.Prev.ID != "a0-greetings" || sess.Next == nil || sess.Next.ID != "a0-sounds-indonesian" {
		t.Fatalf("lesson navigation: prev %v next %v", sess.Prev, sess.Next)
	}
	again, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil || again.ID != sess.ID {
		t.Fatalf("starting twice should resume the same session: %v %s vs %s", err, again.ID, sess.ID)
	}
	if _, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true)); code(err) != "finish_steps_first" {
		t.Fatalf("the checkpoint comes after the steps: %v", err)
	}
	finishSteps(t, s, sess)

	res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed || !res.NewlyMastered || res.Attempt.Score != 100 {
		t.Fatalf("expected pass: %+v", res)
	}
	unlocked := map[string]bool{}
	for _, n := range res.Unlocked {
		unlocked[n.ID] = true
	}
	if !unlocked["a0-sounds-indonesian"] || !unlocked["a0-pronouns"] || !unlocked["a0-common-vocabulary"] || len(unlocked) != 3 {
		t.Fatalf("unexpected unlocks: %v", unlocked)
	}
	if res.Session.Phase != string(tutor.PhaseMastered) || res.Session.EndedAt == nil {
		t.Fatalf("session should be closed as MASTERED, got %s", res.Session.Phase)
	}

	// Repeated completion request: no new attempt, no second mastery.
	again2, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil || !again2.AlreadyCompleted || again2.NewlyMastered {
		t.Fatalf("resubmit should be idempotent: %+v %v", again2, err)
	}
	assertAttempts(t, s, sess.ID, 1)

	// Persistence: a fresh read of the roadmap reflects mastery.
	rm, err := s.Roadmap(ctx, learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]progress.Status{}
	for _, n := range rm.Nodes {
		statuses[n.ID] = n.Status
	}
	if statuses["a0-alphabet-sounds"] != progress.StatusMastered || statuses["a0-pronouns"] != progress.StatusAvailable {
		t.Fatalf("roadmap statuses wrong: %v", statuses)
	}
	if rm.Summary.MasteredNodes != 1 {
		t.Fatalf("mastered count = %d", rm.Summary.MasteredNodes)
	}
}

func assertAttempts(t *testing.T, s *Service, sessionID string, want int) {
	t.Helper()
	var n int
	if err := s.Store.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM assessment_attempts WHERE session_id=$1`, sessionID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("attempts = %d, want %d", n, want)
	}
}

func TestFailedAssessmentReturnsToReviewWithoutUnlocking(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess := startDone(t, s, ctx, "a0-alphabet-sounds")

	res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", false))
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed || res.NewlyMastered || len(res.Unlocked) != 0 || res.Attempt.Score >= 80 {
		t.Fatalf("expected failing result: %+v", res)
	}
	if res.Session.Phase != string(tutor.PhaseLearning) || !res.Session.CanChat || len(res.Session.Assessment) == 0 {
		t.Fatalf("a failed quiz keeps the lesson open for more learning: %+v", res.Session)
	}
	if a := res.Session.LastAttempt; a == nil || a.Passed || res.Session.Lesson.Summary != nil {
		t.Fatalf("a failed checkpoint is recorded without a lesson summary: %+v", res.Session.LastAttempt)
	}
	if res.Session.Node.Status != progress.StatusInProgress {
		t.Fatalf("node status = %s", res.Session.Node.Status)
	}
	if _, err := s.StartSession(ctx, learnerID, course, "a0-pronouns"); code(err) != "node_locked" {
		t.Fatalf("successor must stay locked, got %v", err)
	}

	// The final quiz can be retaken straight away.
	sess = res.Session
	res, err = s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil || !res.Passed || !res.NewlyMastered {
		t.Fatalf("retake should pass: %+v %v", res, err)
	}
	assertAttempts(t, s, sess.ID, 2)
}

func TestCannotStartLockedOrComingSoon(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	if _, err := s.StartSession(ctx, learnerID, course, "a1-to-be"); code(err) != "node_locked" {
		t.Fatalf("locked: got %v", err)
	}
	if _, err := s.StartSession(ctx, learnerID, course, "a1-adjectives"); code(err) != "node_coming_soon" && code(err) != "node_locked" {
		t.Fatalf("coming soon: got %v", err)
	}
	if _, err := s.StartSession(ctx, learnerID, course, "nope"); code(err) != "node_not_found" {
		t.Fatalf("unknown: got %v", err)
	}
	d, err := s.Node(ctx, learnerID, course, "a1-adjectives")
	if err != nil || d.CanStart || d.Status != progress.StatusComingSoon {
		t.Fatalf("coming soon detail: %+v %v", d, err)
	}
}

func TestProviderFailuresNeverPersistOrAwardMastery(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")

	// Tutor unavailable or malformed: nothing stored, retry works.
	for _, fn := range []func(ai.AskInput) (json.RawMessage, error){
		func(ai.AskInput) (json.RawMessage, error) { return nil, ai.ErrUnavailable },
		func(ai.AskInput) (json.RawMessage, error) { return json.RawMessage(`{"answer":""}`), nil },
	} {
		mock.AskFunc = fn
		if _, err := s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: "hello"}); code(err) != "tutor_unavailable" {
			t.Fatalf("got %v", err)
		}
	}
	view, _ := s.GetSession(ctx, learnerID, sess.ID)
	if len(view.Lesson.Steps[0].Asks) != 0 {
		t.Fatalf("failed turns must not be stored: %+v", view.Lesson.Steps[0].Asks)
	}
	mock.AskFunc = nil
	sess = finishSteps(t, s, view)

	for name, fn := range map[string]func(ai.EvalInput) (json.RawMessage, error){
		"timeout": func(ai.EvalInput) (json.RawMessage, error) { return nil, fmt.Errorf("%w: deadline", ai.ErrUnavailable) },
		"malformed": func(ai.EvalInput) (json.RawMessage, error) {
			return json.RawMessage(`{"objectiveScores": "great"}`), nil
		},
		"missing objective": func(ai.EvalInput) (json.RawMessage, error) {
			return json.RawMessage(`{"objectiveScores":{"vowels":100},"evidence":[],"feedback":"x"}`), nil
		},
		"inflated score": func(ai.EvalInput) (json.RawMessage, error) {
			return json.RawMessage(`{"objectiveScores":{"letter-names":500,"vowels":100,"spelling":100},"evidence":[],"feedback":"x"}`), nil
		},
	} {
		mock.EvalFunc = fn
		// Wrong answers, so the evaluator is consulted (exact matches aren't).
		_, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", false))
		var le *Error
		if !errors.As(err, &le) || le.Status != 502 || !le.Retryable {
			t.Fatalf("%s: want retryable 502, got %v", name, err)
		}
	}
	assertAttempts(t, s, sess.ID, 0)
	view, _ = s.GetSession(ctx, learnerID, sess.ID)
	if view.Phase != string(tutor.PhaseLearning) || view.Node.Status == progress.StatusMastered {
		t.Fatalf("failures must not change progress: %s %s", view.Phase, view.Node.Status)
	}
}

func TestConcurrentSubmissionsMasterOnce(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess := startDone(t, s, ctx, "a0-alphabet-sounds")

	const n = 5
	var wg sync.WaitGroup
	results := make([]AssessmentResult, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true))
		}()
	}
	wg.Wait()
	newly := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("submission %d: %v", i, errs[i])
		}
		if results[i].NewlyMastered {
			newly++
		}
	}
	if newly != 1 {
		t.Fatalf("newly mastered %d times, want exactly 1", newly)
	}
	assertAttempts(t, s, sess.ID, 1)
}

func TestFiveLessonsEndToEnd(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	for _, id := range []string{"a0-alphabet-sounds", "a0-pronouns", "a0-common-vocabulary", "a0-to-be-basics", "a0-basic-sentences", "a1-to-be", "a1-simple-present", "a1-questions"} {
		sess := startDone(t, s, ctx, id)
		res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, id, true))
		if err != nil || !res.NewlyMastered {
			t.Fatalf("%s: %+v %v", id, res, err)
		}
	}
	p, err := s.CourseProgress(ctx, learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	// Ready A0 lessons (greetings, numbers, …) are still open, so the learner is A0.
	if len(p.Mastered) != 8 || p.Course.Progress.CurrentLevel != "A0" {
		t.Fatalf("progress: %d mastered, level %s", len(p.Mastered), p.Course.Progress.CurrentLevel)
	}
	d, err := s.Dashboard(ctx, storage.User{ID: learnerID})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Courses) != 1 || d.TotalMastered != 8 || d.Weekly.Steps == 0 || d.Streak.Current != 1 || !d.Streak.ActiveToday || len(d.Activity) != 28 {
		t.Fatalf("dashboard: %+v", d)
	}
}

func TestUsersAreIsolated(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	other := newUser(t, s, "other@example.com")

	if _, err := s.GetSession(ctx, other, sess.ID); code(err) != "session_not_found" {
		t.Fatalf("other user read the session: %v", err)
	}
	if _, err := s.AskStep(ctx, other, sess.ID, 0, AskInput{Content: "hi"}); code(err) != "session_not_found" {
		t.Fatalf("other user wrote to the session: %v", err)
	}
	if _, err := s.NextStep(ctx, other, sess.ID, 0); code(err) != "session_not_found" {
		t.Fatalf("other user moved the session: %v", err)
	}
	if _, err := s.SubmitAssessment(ctx, other, sess.ID, nil); code(err) != "session_not_found" {
		t.Fatalf("other user submitted an assessment: %v", err)
	}
	rm, err := s.Roadmap(ctx, other, course)
	if err != nil {
		t.Fatal(err)
	}
	if rm.Enrolled || rm.Summary.MasteredNodes != 0 {
		t.Fatalf("other user sees progress: %+v", rm.Summary)
	}
	d, err := s.Dashboard(ctx, storage.User{ID: other})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Courses) != 0 || len(d.OpenSessions) != 0 || len(d.Explore) == 0 {
		t.Fatalf("new user's dashboard should be empty with courses to explore: %+v", d)
	}
}

func TestEnrollAndCatalog(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	anon, err := s.Courses(ctx, "")
	if err != nil || len(anon) == 0 || anon[0].Enrolled || anon[0].Progress != nil {
		t.Fatalf("anonymous catalog: %+v %v", anon, err)
	}
	rm, err := s.Enroll(ctx, learnerID, course)
	if err != nil || !rm.Enrolled {
		t.Fatalf("enroll: %+v %v", rm, err)
	}
	if _, err := s.Enroll(ctx, learnerID, course); err != nil {
		t.Fatalf("enroll should be idempotent: %v", err)
	}
	mine, _ := s.Courses(ctx, learnerID)
	if !mine[0].Enrolled || mine[0].Progress == nil || mine[0].Progress.NextNode == nil {
		t.Fatalf("enrolled catalog card: %+v", mine[0])
	}
	if _, err := s.Roadmap(ctx, learnerID, "klingon"); code(err) != "course_not_found" {
		t.Fatalf("unknown course: %v", err)
	}
}

func TestMigrationKeepsV1Data(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", rand.Uint64())
	admin, err := storage.Open(ctx, url, "")
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	store, err := storage.Open(ctx, url, schema)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.MigrateUpTo(ctx, "0001_init.sql"); err != nil {
		t.Fatal(err)
	}
	// v1 data: the demo learner with one mastered node and an open session.
	const demo = "00000000-0000-4000-8000-000000000001"
	for _, q := range []string{
		`INSERT INTO learners (id, display_name, current_level) VALUES ('` + demo + `', 'Learner', 'A1')`,
		`INSERT INTO node_progress (learner_id, node_id, status, mastery_score) VALUES ('` + demo + `', 'a0-alphabet-sounds', 'mastered', 95)`,
		`INSERT INTO learning_sessions (learner_id, node_id, phase) VALUES ('` + demo + `', 'a0-pronouns', 'EXPLANATION')`,
	} {
		if _, err := store.Pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	e, err := storage.GetEnrollment(ctx, store.Pool, demo, "english")
	if err != nil || e.CurrentLevel != "A1" || e.TargetLevel != "B2" {
		t.Fatalf("enrollment not backfilled: %+v %v", e, err)
	}
	rows, err := storage.ListProgress(ctx, store.Pool, demo, "english")
	if err != nil || len(rows) != 1 || rows[0].Status != "mastered" {
		t.Fatalf("progress not kept: %+v %v", rows, err)
	}
	if _, err := storage.FindOpenSession(ctx, store.Pool, demo, "english", "a0-pronouns"); err != nil {
		t.Fatalf("open session not kept: %v", err)
	}
}

func TestStreak(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, loc)
	days := map[string]int{"2026-10-13": 2, "2026-10-12": 1, "2026-10-10": 4}
	grid, st := activity(days, now)
	if st.Current != 2 || st.ActiveToday || len(grid) != 28 || grid[27].Date != "2026-10-14" || grid[26].Count != 2 {
		t.Fatalf("yesterday-anchored streak: %+v %+v", st, grid[25:])
	}
	days["2026-10-14"] = 1
	if _, st = activity(days, now); st.Current != 3 || !st.ActiveToday {
		t.Fatalf("today-anchored streak: %+v", st)
	}
	if _, st = activity(map[string]int{"2026-10-11": 1}, now); st.Current != 0 {
		t.Fatalf("broken streak should be 0: %+v", st)
	}
}

func TestLessonKeepsItsLanguage(t *testing.T) {
	var seen []ai.AskInput
	mock := &ai.Mock{}
	inner := &ai.Mock{}
	mock.AskFunc = func(in ai.AskInput) (json.RawMessage, error) {
		seen = append(seen, in)
		r, err := inner.Ask(context.Background(), in)
		return r.Raw, err
	}
	s := newService(t, mock)
	idCtx := WithLang(context.Background(), "id")
	sess, err := s.StartSession(idCtx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if sess.Node.Title != "Alfabet & Bunyi" || sess.Course.Title != "Bahasa Inggris" || sess.Material.Sections[0].Title != "Nama huruf berbeda dari bahasa Indonesia" {
		t.Fatalf("views should be localized: %q %q", sess.Node.Title, sess.Course.Title)
	}
	cur, _ := s.Catalog.Course(course)
	if want, _, _ := tutor.AuthoredPlan(cur, "a0-alphabet-sounds", "id"); sess.Lesson.Steps[0].Title != want.Steps[0].Title {
		t.Fatalf("the plan should be in the lesson's language: %q", sess.Lesson.Steps[0].Title)
	}
	// Switching the UI language later doesn't switch a lesson mid-way.
	if _, err := s.AskStep(WithLang(context.Background(), "en"), learnerID, sess.ID, 0, AskInput{Content: "Kenapa I dibaca ai?"}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].Locale != "id" || !strings.Contains(seen[0].LanguagePolicy, "Indonesian") {
		t.Fatalf("the tutor should answer in the lesson's language: %+v", seen)
	}
}

// startDone starts a lesson and walks all of its steps, so the checkpoint is open.
func startDone(t *testing.T, s *Service, ctx context.Context, nodeID string) SessionView {
	t.Helper()
	sess, err := s.StartSession(ctx, learnerID, course, nodeID)
	if err != nil {
		t.Fatalf("%s: %v", nodeID, err)
	}
	return finishSteps(t, s, sess)
}

// planOf reads the server-side plan (with answers) of a session.
func planOf(t *testing.T, s *Service, sessionID string) tutor.Plan {
	t.Helper()
	sess, err := storage.GetSession(context.Background(), s.Store.Pool, sessionID, learnerID)
	if err != nil {
		t.Fatal(err)
	}
	var p tutor.Plan
	if err := json.Unmarshal(sess.Plan, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// rightAnswer is the correct answer to q.
func rightAnswer(q tutor.Question) AnswerInput {
	if q.Type == tutor.QuestionMCQ {
		a := q.Answer
		return AnswerInput{Choice: &a}
	}
	return AnswerInput{Tokens: strings.Fields(q.Solution)}
}

// wrongAnswer is an incorrect answer to q.
func wrongAnswer(q tutor.Question) AnswerInput {
	if q.Type == tutor.QuestionMCQ {
		a := (q.Answer + 1) % len(q.Options)
		return AnswerInput{Choice: &a}
	}
	w := strings.Fields(q.Solution)
	w[0], w[len(w)-1] = w[len(w)-1], w[0]
	return AnswerInput{Tokens: w}
}

// finishSteps answers every open question correctly and moves to the checkpoint.
func finishSteps(t *testing.T, s *Service, sess SessionView) SessionView {
	t.Helper()
	ctx := context.Background()
	plan := planOf(t, s, sess.ID)
	for !sess.Lesson.Checkpoint {
		i := sess.Lesson.StepIndex
		var err error
		if q := sess.Lesson.Steps[i].Question; q != nil && !q.Answered && !q.Revealed {
			if sess, err = s.AnswerStep(ctx, learnerID, sess.ID, i, rightAnswer(*plan.Steps[i].Question)); err != nil {
				t.Fatalf("answer step %d: %v", i, err)
			}
		}
		if sess, err = s.NextStep(ctx, learnerID, sess.ID, i); err != nil {
			t.Fatalf("next step %d: %v", i, err)
		}
	}
	return sess
}
