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

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/progress"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

const learnerID = "00000000-0000-4000-8000-0000000000aa"

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
	if err := store.EnsureLearner(ctx, learnerID, "Test"); err != nil {
		t.Fatal(err)
	}
	cur, err := curriculum.Load("../../../../packages/curriculum")
	if err != nil {
		t.Fatal(err)
	}
	return &Service{Store: store, Cur: cur, AI: provider, LearnerID: learnerID}
}

func code(err error) string {
	var le *Error
	if errors.As(err, &le) {
		return le.Code
	}
	return ""
}

// driveToAssessment answers practice prompts correctly until the session
// reaches ASSESSMENT.
func driveToAssessment(t *testing.T, s *Service, sess SessionView) SessionView {
	t.Helper()
	ctx := context.Background()
	lesson, _ := s.Cur.Lesson(sess.NodeID)
	prompts := append(append([]curriculum.PracticePrompt{}, lesson.GuidedPractice...), lesson.IndependentPrompt...)
	for i := 0; i < 40 && sess.Phase != string(tutor.PhaseAssessment); i++ {
		// Answer whichever practice prompt the tutor asked last, like a learner would.
		answer := "ready"
		last := sess.Messages[len(sess.Messages)-1].Content
		at := -1
		for _, p := range prompts {
			if k := strings.LastIndex(last, p.Prompt); k > at {
				answer, at = p.Expected[0], k
			}
		}
		var err error
		sess, err = s.SendMessage(ctx, sess.ID, answer)
		if err != nil {
			t.Fatalf("send message in %s: %v", sess.Phase, err)
		}
	}
	if sess.Phase != string(tutor.PhaseAssessment) {
		t.Fatalf("never reached assessment, stuck in %s", sess.Phase)
	}
	return sess
}

func answers(s *Service, nodeID string, correct bool) map[string]string {
	lesson, _ := s.Cur.Lesson(nodeID)
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

	sess, err := s.StartSession(ctx, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if len(sess.Messages) != 1 || sess.Messages[0].Role != "assistant" {
		t.Fatalf("expected the intro message, got %+v", sess.Messages)
	}
	again, err := s.StartSession(ctx, "a0-alphabet-sounds")
	if err != nil || again.ID != sess.ID {
		t.Fatalf("starting twice should resume the same session: %v %s vs %s", err, again.ID, sess.ID)
	}

	sess = driveToAssessment(t, s, sess)
	if len(sess.Assessment) == 0 {
		t.Fatal("assessment items should be exposed in ASSESSMENT")
	}

	res, err := s.SubmitAssessment(ctx, sess.ID, answers(s, "a0-alphabet-sounds", true))
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
	if !unlocked["a0-pronouns"] || !unlocked["a0-common-vocabulary"] || len(unlocked) != 2 {
		t.Fatalf("unexpected unlocks: %v", unlocked)
	}
	if res.Session.Phase != string(tutor.PhaseMastered) || res.Session.EndedAt == nil {
		t.Fatalf("session should be closed as MASTERED, got %s", res.Session.Phase)
	}

	// Repeated completion request: no new attempt, no second mastery.
	again2, err := s.SubmitAssessment(ctx, sess.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil || !again2.AlreadyCompleted || again2.NewlyMastered {
		t.Fatalf("resubmit should be idempotent: %+v %v", again2, err)
	}
	assertAttempts(t, s, sess.ID, 1)

	// Persistence: a fresh read of the roadmap reflects mastery.
	rm, err := s.Roadmap(ctx)
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
	sess, _ := s.StartSession(ctx, "a0-alphabet-sounds")
	sess = driveToAssessment(t, s, sess)

	res, err := s.SubmitAssessment(ctx, sess.ID, answers(s, "a0-alphabet-sounds", false))
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed || res.NewlyMastered || len(res.Unlocked) != 0 || res.Attempt.Score >= 80 {
		t.Fatalf("expected failing result: %+v", res)
	}
	if res.Session.Phase != string(tutor.PhaseReview) {
		t.Fatalf("phase = %s, want REVIEW", res.Session.Phase)
	}
	if !res.Session.CanChat {
		t.Fatal("learner should be able to chat during review")
	}
	if res.Session.Node.Status != progress.StatusInProgress {
		t.Fatalf("node status = %s", res.Session.Node.Status)
	}
	if _, err := s.StartSession(ctx, "a0-pronouns"); code(err) != "node_locked" {
		t.Fatalf("successor must stay locked, got %v", err)
	}

	// Review leads back to a retake, which can pass.
	sess = driveToAssessment(t, s, res.Session)
	res, err = s.SubmitAssessment(ctx, sess.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil || !res.Passed || !res.NewlyMastered {
		t.Fatalf("retake should pass: %+v %v", res, err)
	}
	assertAttempts(t, s, sess.ID, 2)
}

func TestCannotStartLockedOrComingSoon(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	if _, err := s.StartSession(ctx, "a1-to-be"); code(err) != "node_locked" {
		t.Fatalf("locked: got %v", err)
	}
	if _, err := s.StartSession(ctx, "a0-greetings"); code(err) != "node_coming_soon" && code(err) != "node_locked" {
		t.Fatalf("coming soon: got %v", err)
	}
	if _, err := s.StartSession(ctx, "nope"); code(err) != "node_not_found" {
		t.Fatalf("unknown: got %v", err)
	}
	d, err := s.Node(ctx, "a0-greetings")
	if err != nil || d.CanStart || d.Status != progress.StatusComingSoon {
		t.Fatalf("coming soon detail: %+v %v", d, err)
	}
}

func TestMessageValidation(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, "a0-alphabet-sounds")
	if _, err := s.SendMessage(ctx, sess.ID, "   "); code(err) != "empty_message" {
		t.Fatalf("empty: %v", err)
	}
	if _, err := s.SendMessage(ctx, sess.ID, strings.Repeat("a", MaxMessageRunes+1)); code(err) != "message_too_long" {
		t.Fatalf("long: %v", err)
	}
	if _, err := s.SendMessage(ctx, "00000000-0000-4000-8000-00000000ffff", "hi"); code(err) != "session_not_found" {
		t.Fatalf("missing session: %v", err)
	}
	if _, err := s.SubmitAssessment(ctx, sess.ID, answers(s, "a0-alphabet-sounds", true)); code(err) != "wrong_phase" {
		t.Fatalf("assessment before ready: %v", err)
	}
}

func TestProviderFailuresNeverPersistOrAwardMastery(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, "a0-alphabet-sounds")

	// Tutor unavailable: nothing stored, retry works.
	mock.TutorFunc = func(ai.TutorInput) (json.RawMessage, error) { return nil, ai.ErrUnavailable }
	if _, err := s.SendMessage(ctx, sess.ID, "hello"); code(err) != "tutor_unavailable" {
		t.Fatalf("got %v", err)
	}
	// Malformed tutor output: also rejected.
	mock.TutorFunc = func(ai.TutorInput) (json.RawMessage, error) { return json.RawMessage(`{"message":""}`), nil }
	if _, err := s.SendMessage(ctx, sess.ID, "hello"); code(err) != "tutor_unavailable" {
		t.Fatalf("got %v", err)
	}
	view, _ := s.GetSession(ctx, sess.ID)
	if len(view.Messages) != 1 || view.PhaseTurns != 0 {
		t.Fatalf("failed turns must not be stored: %d messages, %d turns", len(view.Messages), view.PhaseTurns)
	}
	mock.TutorFunc = nil
	sess = driveToAssessment(t, s, view)

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
		_, err := s.SubmitAssessment(ctx, sess.ID, answers(s, "a0-alphabet-sounds", true))
		var le *Error
		if !errors.As(err, &le) || le.Status != 502 || !le.Retryable {
			t.Fatalf("%s: want retryable 502, got %v", name, err)
		}
	}
	assertAttempts(t, s, sess.ID, 0)
	view, _ = s.GetSession(ctx, sess.ID)
	if view.Phase != string(tutor.PhaseAssessment) || view.Node.Status == progress.StatusMastered {
		t.Fatalf("failures must not change progress: %s %s", view.Phase, view.Node.Status)
	}
}

func TestConcurrentSubmissionsMasterOnce(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, "a0-alphabet-sounds")
	sess = driveToAssessment(t, s, sess)

	const n = 5
	var wg sync.WaitGroup
	results := make([]AssessmentResult, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = s.SubmitAssessment(ctx, sess.ID, answers(s, "a0-alphabet-sounds", true))
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
	for _, id := range []string{"a0-alphabet-sounds", "a0-pronouns", "a0-common-vocabulary", "a0-basic-sentences", "a1-to-be", "a1-simple-present", "a1-questions"} {
		sess, err := s.StartSession(ctx, id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		sess = driveToAssessment(t, s, sess)
		res, err := s.SubmitAssessment(ctx, sess.ID, answers(s, id, true))
		if err != nil || !res.NewlyMastered {
			t.Fatalf("%s: %+v %v", id, res, err)
		}
	}
	p, err := s.Progress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Mastered) != 7 || p.Learner.CurrentLevel != "A1" {
		t.Fatalf("progress: %d mastered, level %s", len(p.Mastered), p.Learner.CurrentLevel)
	}
}
