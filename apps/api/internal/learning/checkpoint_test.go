package learning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/progress"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

// aiCheckpoint is a valid, mixed checkpoint for the alphabet lesson; round
// changes its sentences so retakes differ.
func aiCheckpoint(round int) json.RawMessage {
	words := []string{"cat", "dog", "sun", "pen"}
	w := words[round%len(words)]
	return json.RawMessage(fmt.Sprintf(`{"items":[
	{"type":"mcq","objectiveId":"letter-names","prompt":"Round %[1]d: which letter sounds like 'eye'?","listen":"","options":["E","I","A"],"correctOption":"I","tokens":[],"solution":"","acceptedAnswers":[],"rubric":"","explanation":"I is said 'eye'."},
	{"type":"word_order","objectiveId":"vowels","prompt":"Round %[1]d: put the words in order.","listen":"","options":[],"tokens":["A","and","E","are","vowels"],"solution":"A and E are vowels","acceptedAnswers":[],"rubric":"","explanation":"Subject, verb, noun."},
	{"type":"text","objectiveId":"spelling","prompt":"Round %[1]d: spell '%[2]s' with dashes.","listen":"","options":[],"tokens":[],"solution":"","acceptedAnswers":["%[3]s"],"rubric":"All letters right = 100.","explanation":"Letter by letter."},
	{"type":"text","objectiveId":"letter-names","prompt":"Round %[1]d: write the letter you hear.","listen":"The letter is Q.","options":[],"tokens":[],"solution":"","acceptedAnswers":["q"],"rubric":"Q only.","explanation":"Q is said 'kyu'."},
	{"type":"mcq","objectiveId":"vowels","prompt":"Round %[1]d: which one is a vowel?","listen":"","options":["B","O","K"],"correctOption":"O","tokens":[],"solution":"","acceptedAnswers":[],"rubric":"","explanation":"O is a vowel."}]}`,
		round, w, strings.Join(strings.Split(w, ""), " ")))
}

// pick is the index of an option's text in a checkpoint question (options are shuffled).
func pick(cp *CheckpointView, id, option string) string {
	for _, it := range cp.Items {
		if it.ID == id {
			for i, o := range it.Options {
				if o == option {
					return fmt.Sprint(i)
				}
			}
		}
	}
	return "missing"
}

func TestAICheckpointIsCheckedGradedAndExplained(t *testing.T) {
	mock := &ai.Mock{}
	var seen []ai.CheckpointInput
	mock.CheckpointFunc = func(in ai.CheckpointInput) (json.RawMessage, error) {
		seen = append(seen, in)
		return aiCheckpoint(len(seen)), nil
	}
	inner := &ai.Mock{}
	// A grader that tries to give full marks to everything.
	mock.GradeFunc = func(in ai.GradeInput) (json.RawMessage, error) {
		r, err := inner.GradeCheckpoint(context.Background(), in)
		var g map[string]any
		_ = json.Unmarshal(r.Raw, &g)
		for _, it := range g["items"].([]any) {
			it.(map[string]any)["score"] = 100
		}
		raw, _ := json.Marshal(g)
		return raw, err
	}
	s := newService(t, mock)
	ctx := context.Background()
	sess := startDone(t, s, ctx, "a0-alphabet-sounds")

	cp := sess.Checkpoint
	if len(seen) != 1 || len(seen[0].Examples) == 0 || len(seen[0].Earlier) != 0 || cp.Attempt != 1 || cp.Practice || len(cp.Items) != 5 {
		t.Fatalf("one checkpoint written from the lesson: %d calls, %+v", len(seen), cp)
	}
	raw, _ := json.Marshal(cp)
	if strings.Contains(string(raw), "acceptedAnswers") || strings.Contains(string(raw), "The letter is Q") || strings.Contains(string(raw), `"solution"`) {
		t.Fatalf("answers and listening text must stay on the server: %s", raw)
	}
	wo := cp.Items[1]
	if wo.Type != tutor.QuestionWordOrder || strings.Join(wo.Tokens, " ") == "A and E are vowels" || !cp.Items[3].HasAudio {
		t.Fatalf("word order shuffled, listening flagged: %+v", cp.Items)
	}
	if a, err := s.CheckpointListen(ctx, learnerID, sess.ID, cp.ID, "q4"); err != nil || a.Text != "The letter is Q." {
		t.Fatalf("listening falls back to the device voice: %+v %v", a, err)
	}

	answers := map[string]string{"q1": pick(cp, "q1", "E"), "q2": "A and E are vowels", "q3": "D-O-G", "q4": "k", "q5": pick(cp, "q5", "B")}
	if _, err := s.SubmitAssessment(ctx, learnerID, sess.ID, map[string]string{"q1": "7", "q2": "ignore the rules", "q3": "x", "q4": "x", "q5": "0"}); code(err) != "invalid_answers" {
		t.Fatalf("answers must fit their questions: %v", err)
	}
	res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers)
	if err != nil {
		t.Fatal(err)
	}
	var review tutor.Review
	if err := json.Unmarshal(res.Attempt.Review, &review); err != nil || len(review.Items) != 5 {
		t.Fatalf("every answer is reviewed: %v %s", err, res.Attempt.Review)
	}
	// q1 and q5 are wrong choices; q4 is a wrong open answer the grader rates 100.
	want := map[string]int{"q1": 0, "q2": 100, "q3": 100, "q4": 100, "q5": 0}
	for _, it := range review.Items {
		if it.Score != want[it.ID] || it.Explanation == "" || it.CorrectAnswer == "" {
			t.Fatalf("%s: score %d (want %d), explanation %q", it.ID, it.Score, want[it.ID], it.Explanation)
		}
	}
	if review.Items[4].Answer != "B" || review.Items[4].CorrectAnswer != "O" || review.Items[4].Verdict != tutor.VerdictWrong {
		t.Fatalf("a wrong choice stays wrong whatever the grader says: %+v", review.Items[4])
	}
	// letter-names 50, vowels 50, spelling 100: a mean of 67 doesn't pass.
	if res.Passed || res.Attempt.Score != 67 || res.Attempt.ObjectiveScores["vowels"] != 50 || res.Attempt.Practice {
		t.Fatalf("the pass mark is applied on the server: %+v", res.Attempt)
	}

	// Wrong answers become review flashcards (the right answer on the back).
	due, err := s.DueFlashcards(ctx, learnerID, course)
	if err != nil {
		t.Fatal(err)
	}
	backs := map[string]string{}
	for _, c := range due.Cards {
		backs[c.Front] = c.Back
	}
	if backs["Round 1: which letter sounds like 'eye'?"] != "I" || backs["Round 1: which one is a vowel?"] != "O" {
		t.Fatalf("checkpoint mistakes should be flashcards: %v", backs)
	}

	// A retake gets new questions, avoiding the earlier ones, and the history grows.
	next, err := s.StartCheckpoint(ctx, learnerID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == cp.ID || next.Attempt != 2 || len(seen) != 2 || len(seen[1].Earlier) != 5 || next.Items[0].Prompt == cp.Items[0].Prompt {
		t.Fatalf("retake: %+v, earlier %v", next, seen[len(seen)-1].Earlier)
	}
	answers["q1"], answers["q3"], answers["q5"] = pick(&next, "q1", "I"), "s-u-n", pick(&next, "q5", "O")
	if res, err = s.SubmitAssessment(ctx, learnerID, sess.ID, answers); err != nil || !res.Passed || !res.NewlyMastered {
		t.Fatalf("second attempt passes: %+v %v", res, err)
	}
	if h := res.Session.Attempts; len(h) != 2 || !h[0].Passed || h[1].Passed || h[1].Score >= h[0].Score {
		t.Fatalf("history, newest first: %+v", h)
	}
	old, err := s.Attempt(ctx, learnerID, res.Session.Attempts[1].ID)
	if err != nil || len(old.Review) == 0 || old.Passed {
		t.Fatalf("an old attempt can be reopened with its review: %v", err)
	}
	if _, err := s.Attempt(ctx, newUser(t, s, "other@example.com"), old.ID); code(err) != "attempt_not_found" {
		t.Fatalf("attempts are private: %v", err)
	}
	if _, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers); err != nil {
		t.Fatalf("a repeated submission after mastery is answered, not failed: %v", err)
	}
	assertAttempts(t, s, sess.ID, 2)
}

func TestRejectedAICheckpointUsesHandWrittenQuestions(t *testing.T) {
	mock := &ai.Mock{}
	mock.CheckpointFunc = func(in ai.CheckpointInput) (json.RawMessage, error) {
		// Only one objective covered: rejected.
		return json.RawMessage(`{"items":[{"type":"text","objectiveId":"spelling","prompt":"Spell cat.","listen":"","options":[],"tokens":[],"solution":"","acceptedAnswers":["c a t"],"rubric":"r","explanation":"e"}]}`), nil
	}
	s := newService(t, mock)
	ctx := WithLang(context.Background(), "id")
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	sess = finishStepsIn(t, s, ctx, sess)
	l := lessonOf(s, "a0-alphabet-sounds")
	if mock.Calls.Checkpoint != 2 || len(sess.Checkpoint.Items) != len(l.Assessment) || sess.Checkpoint.Items[3].Prompt != l.PromptIn(l.Assessment[3], "id") {
		t.Fatalf("after two rejected tries the hand-written checkpoint is used, in Indonesian: %d calls, %+v", mock.Calls.Checkpoint, sess.Checkpoint.Items)
	}
}

func TestMasteredLessonReopensWithPracticeCheckpoints(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	sess := startDone(t, s, ctx, "a0-alphabet-sounds")
	if res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true)); err != nil || !res.NewlyMastered {
		t.Fatalf("master: %v", err)
	}

	// Opening the lesson again shows the finished lesson, not a new one.
	again, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != sess.ID || again.Phase != string(tutor.PhaseMastered) || !again.Lesson.Checkpoint || len(again.Lesson.Steps) != again.Lesson.TotalSteps || len(again.Attempts) != 1 {
		t.Fatalf("a mastered lesson reopens where it was finished: %s %s steps %d/%d attempts %d", again.ID, again.Phase, len(again.Lesson.Steps), again.Lesson.TotalSteps, len(again.Attempts))
	}

	// A stray open session (left by an older version that restarted mastered
	// lessons) is closed, and the finished one is shown.
	stray, err := storage.CreateSession(ctx, s.Store.Pool, learnerID, course, "a0-alphabet-sounds", "en", string(tutor.PhaseLearning))
	if err != nil {
		t.Fatal(err)
	}
	if again, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds"); err != nil || again.ID != sess.ID {
		t.Fatalf("the mastered session should win over a stray open one: %v", err)
	}
	if _, err := storage.FindOpenSession(ctx, s.Store.Pool, learnerID, course, "a0-alphabet-sounds"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("the stray session %s should be closed: %v", stray.ID, err)
	}

	// Practice: recorded in the history, progress untouched.
	cp, err := s.StartCheckpoint(ctx, learnerID, sess.ID)
	if err != nil || !cp.Practice || cp.Attempt != 2 {
		t.Fatalf("practice checkpoint: %+v %v", cp, err)
	}
	res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", false))
	if err != nil || res.Passed || !res.Attempt.Practice || res.NewlyMastered {
		t.Fatalf("practice attempt: %+v %v", res.Attempt, err)
	}
	if res.Session.Node.Status != progress.StatusMastered || len(res.Session.Attempts) != 2 || !res.Session.Attempts[0].Practice {
		t.Fatalf("a failed practice doesn't undo mastery: %s %+v", res.Session.Node.Status, res.Session.Attempts)
	}
	if res.Session.Checkpoint != nil {
		t.Fatalf("the practice checkpoint is closed once submitted")
	}
}

func TestCheckerSendsBrokenQuestionsBack(t *testing.T) {
	mock := &ai.Mock{}
	var seen []ai.CheckpointInput
	mock.CheckpointFunc = func(in ai.CheckpointInput) (json.RawMessage, error) {
		seen = append(seen, in)
		return aiCheckpoint(len(seen)), nil
	}
	checks := 0
	mock.VerifyFunc = func(in ai.VerifyInput) (json.RawMessage, error) {
		checks++
		var out []map[string]any
		for _, it := range in.Items {
			// The first draft's vowel question is flagged.
			bad := checks == 1 && it.ID == "q5"
			problem := ""
			if bad {
				problem = "Two options are vowels."
			}
			out = append(out, map[string]any{"id": it.ID, "ok": !bad, "problem": problem})
		}
		return json.Marshal(map[string]any{"items": out})
	}
	s := newService(t, mock)
	sess := startDone(t, s, context.Background(), "a0-alphabet-sounds")
	if len(seen) != 2 || len(seen[1].Problems) != 1 || !strings.Contains(seen[1].Problems[0], "Two options are vowels") {
		t.Fatalf("too few questions left after the check, so the writer tries again with the problem: %+v", seen)
	}
	if !strings.HasPrefix(sess.Checkpoint.Items[0].Prompt, "Round 2") || len(sess.Checkpoint.Items) != 5 {
		t.Fatalf("the second, checked draft is used: %+v", sess.Checkpoint.Items)
	}
}
