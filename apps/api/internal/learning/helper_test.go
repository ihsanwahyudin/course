package learning

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/resources"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

func lastMeta(t *testing.T, v SessionView) msgMeta {
	t.Helper()
	var m msgMeta
	if err := json.Unmarshal(v.Messages[len(v.Messages)-1].Metadata, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHelperQuizIsCachedAcrossLearners(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := context.Background()
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Intent: "quiz"})
	if err != nil {
		t.Fatal(err)
	}
	m := lastMeta(t, v)
	if len(m.Quiz) < 2 || m.Cached || v.Messages[len(v.Messages)-2].Content != "Give me a practice quiz." {
		t.Fatalf("first quiz: %+v / %q", m, v.Messages[len(v.Messages)-2].Content)
	}
	if mock.Calls.Tutor != 1 {
		t.Fatalf("tutor calls = %d", mock.Calls.Tutor)
	}

	// Another learner pressing the same button gets the cached quiz for free.
	other := newUser(t, s, "b@example.com")
	sess2, _ := s.StartSession(ctx, other, course, "a0-alphabet-sounds")
	v2, err := s.SendMessage(ctx, other, sess2.ID, MessageInput{Intent: "quiz"})
	if err != nil {
		t.Fatal(err)
	}
	if m2 := lastMeta(t, v2); !m2.Cached || len(m2.Quiz) != len(m.Quiz) || mock.Calls.Tutor != 1 {
		t.Fatalf("second learner should hit the cache: cached=%v calls=%d", m2.Cached, mock.Calls.Tutor)
	}
	// Asking again in the same lesson is a new round: a fresh call.
	if _, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Intent: "quiz"}); err != nil || mock.Calls.Tutor != 2 {
		t.Fatalf("second quiz in a session should be new: calls=%d err=%v", mock.Calls.Tutor, err)
	}
	// Typed questions are never cached.
	if _, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Content: "What is a vowel?"}); err != nil || mock.Calls.Tutor != 3 {
		t.Fatalf("typed question: calls=%d err=%v", mock.Calls.Tutor, err)
	}
}

func TestHelperHistoryIsTrimmed(t *testing.T) {
	var seen []ai.TutorInput
	mock := &ai.Mock{}
	inner := &ai.Mock{}
	mock.TutorFunc = func(in ai.TutorInput) (json.RawMessage, error) {
		seen = append(seen, in)
		r, err := inner.TutorReply(context.Background(), in)
		return r.Raw, err
	}
	s := newService(t, mock)
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if _, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Content: "flashcards please", Intent: "flashcards"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Content: "Tell me about vowels " + strings.Repeat("x", 10)}); err != nil {
			t.Fatal(err)
		}
	}
	last := seen[len(seen)-1]
	if len(last.Messages) != historyMessages+1 || last.Messages[len(last.Messages)-1].Role != "user" {
		t.Fatalf("history should be capped at %d + the new message, got %d", historyMessages, len(last.Messages))
	}
	if last.CacheKey != "english:a0-alphabet-sounds:en" || last.Intent != tutor.IntentAsk {
		t.Fatalf("cache key / intent: %q %q", last.CacheKey, last.Intent)
	}
	// The flashcard reply is summarised, not resent in full.
	h := trimHistory([]storage.Message{{Role: "assistant", Content: strings.Repeat("y", 900), Metadata: json.RawMessage(`{"flashcards":[{"front":"a","back":"b"}]}`)}})
	if len([]rune(h[0].Content)) > historyReplyRunes+40 || !strings.HasSuffix(h[0].Content, "[flashcards: 1]") {
		t.Fatalf("trimmed reply: %q", h[0].Content)
	}
}

func TestDailyTokenBudget(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	s.DailyTokens = 1500 // the mock costs 1000 + 100 per call
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if _, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Content: "one"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Content: "two"}); err != nil {
		t.Fatal(err) // 1100 used, still under
	}
	if _, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Content: "three"}); code(err) != "ai_quota" {
		t.Fatalf("over budget: %v", err)
	}
	u, _ := storage.UsageOn(ctx, s.Store.Pool, learnerID, s.day())
	if u.Calls != 2 || u.Input != 2000 || u.Output != 200 {
		t.Fatalf("usage not recorded: %+v", u)
	}
	// The final quiz still works when every answer matches (no AI needed).
	if res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true)); err != nil || !res.Passed {
		t.Fatalf("quiz over budget: %+v %v", res, err)
	}
}

func TestExactAnswersSkipTheEvaluator(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	res, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-alphabet-sounds", true))
	if err != nil || !res.Passed || res.Attempt.Score != 100 || mock.Calls.Eval != 0 {
		t.Fatalf("exact answers: %+v calls=%d err=%v", res.Attempt, mock.Calls.Eval, err)
	}

	// A wrong answer for one objective: only that objective goes to the AI,
	// and its score blends with the exact matches.
	sess, _ = s.StartSession(ctx, learnerID, course, "a0-pronouns")
	var got ai.EvalInput
	mock.EvalFunc = func(in ai.EvalInput) (json.RawMessage, error) {
		got = in
		return json.RawMessage(`{"objectiveScores":{"possessives":50},"evidence":["half right"],"feedback":"Check her/our."}`), nil
	}
	ans := answers(s, "a0-pronouns", true)
	ans["pron-4"] = "their"
	res, err = s.SubmitAssessment(ctx, learnerID, sess.ID, ans)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "pron-4" || len(got.Node.Objectives) != 1 || got.Node.Objectives[0].ID != "possessives" {
		t.Fatalf("evaluator should see only the unmatched item and objective: %+v", got)
	}
	// possessives: one exact (100) + one AI-scored (50) → 75; overall (100+75+100)/3 = 92.
	if res.Attempt.ObjectiveScores["possessives"] != 75 || res.Attempt.Score != 92 || !res.Passed {
		t.Fatalf("blended scores wrong: %+v", res.Attempt)
	}
}

func TestResourcesCuratedAndImages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"query":{"pages":[{"index":1,"title":"File:Alphabet.jpg","imageinfo":[{"thumburl":"https://upload.wikimedia.org/a.jpg","thumbwidth":480,"thumbheight":320,
			"descriptionurl":"https://commons.wikimedia.org/wiki/File:Alphabet.jpg","mime":"image/jpeg","extmetadata":{"LicenseShortName":{"value":"CC0"}}}]}]}}`)
	}))
	defer srv.Close()
	mock := &ai.Mock{}
	s := newService(t, mock)
	s.Commons = resources.NewCommons("test")
	s.Commons.BaseURL = srv.URL
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")

	// Videos: the mock can't search the web, so the curated video comes back.
	v, err := s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Intent: "video"})
	if err != nil {
		t.Fatal(err)
	}
	m := lastMeta(t, v)
	if len(m.Resources) != 1 || !m.Resources[0].Curated || m.Resources[0].VideoID != "wlbQLBhLk_4" {
		t.Fatalf("curated video expected: %+v", m.Resources)
	}
	// Images come from Commons with a credit line, and are cached.
	v, err = s.SendMessage(ctx, learnerID, sess.ID, MessageInput{Intent: "image"})
	if err != nil {
		t.Fatal(err)
	}
	m = lastMeta(t, v)
	if len(m.Resources) != 1 || m.Resources[0].Thumb != "https://upload.wikimedia.org/a.jpg" || m.Resources[0].License != "CC0" || m.Resources[0].Author == "" {
		t.Fatalf("commons image expected: %+v", m.Resources)
	}
	srv.Close() // a second identical search must come from the cache
	other := newUser(t, s, "c@example.com")
	sess2, _ := s.StartSession(ctx, other, course, "a0-alphabet-sounds")
	v, err = s.SendMessage(ctx, other, sess2.ID, MessageInput{Intent: "image"})
	if err != nil || len(lastMeta(t, v).Resources) != 1 {
		t.Fatalf("cached image search: %v %+v", err, lastMeta(t, v).Resources)
	}
}

func TestFlashcards(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := context.Background()
	lesson := lessonOf(s, "a1-to-be")
	in := FlashcardInput{Course: course, NodeID: "a1-to-be"}
	for _, f := range lesson.Flashcards {
		in.Cards = append(in.Cards, struct {
			Front string `json:"front"`
			Back  string `json:"back"`
		}{f.Front, f.Back})
	}
	res, err := s.SaveFlashcards(ctx, learnerID, in)
	if err != nil || res.Added != len(lesson.Flashcards) || res.Due != res.Added {
		t.Fatalf("save starter deck: %+v %v", res, err)
	}
	if res, _ = s.SaveFlashcards(ctx, learnerID, in); res.Added != 0 {
		t.Fatalf("duplicates must be skipped: %+v", res)
	}
	bad := in
	bad.Cards = append(bad.Cards[:0:0], struct {
		Front string `json:"front"`
		Back  string `json:"back"`
	}{"", "x"})
	if _, err := s.SaveFlashcards(ctx, learnerID, bad); code(err) != "invalid_flashcards" {
		t.Fatalf("empty card: %v", err)
	}

	q, err := s.DueFlashcards(WithLang(ctx, "id"), learnerID, course)
	if err != nil || len(q.Cards) != len(lesson.Flashcards) || q.Cards[0].NodeTitle != "Kata Kerja “To Be”" {
		t.Fatalf("due queue: %+v %v", q, err)
	}
	card := q.Cards[0]
	f, err := s.ReviewFlashcard(ctx, learnerID, card.ID, GradeGood)
	if err != nil || f.IntervalDays != 1 || f.Reps != 1 || time.Until(f.DueAt) < 23*time.Hour {
		t.Fatalf("good review: %+v %v", f, err)
	}
	if _, err := s.ReviewFlashcard(ctx, learnerID, card.ID, "meh"); code(err) != "invalid_grade" {
		t.Fatalf("grade: %v", err)
	}
	other := newUser(t, s, "d@example.com")
	if _, err := s.ReviewFlashcard(ctx, other, card.ID, GradeGood); code(err) != "flashcard_not_found" {
		t.Fatalf("another learner's card: %v", err)
	}
	if err := s.DeleteFlashcard(ctx, other, card.ID); code(err) != "flashcard_not_found" {
		t.Fatalf("delete another learner's card: %v", err)
	}
	if q, _ = s.DueFlashcards(ctx, learnerID, ""); len(q.Cards) != len(lesson.Flashcards)-1 || q.Total != len(lesson.Flashcards) {
		t.Fatalf("reviewed card should leave the queue: %+v", q)
	}
	d, _ := s.Dashboard(ctx, storage.User{ID: learnerID})
	if d.Flashcards.Due != len(lesson.Flashcards)-1 || d.Flashcards.Total != len(lesson.Flashcards) {
		t.Fatalf("dashboard flashcards: %+v", d.Flashcards)
	}
}

func TestSchedule(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	f := schedule(storage.Flashcard{}, GradeGood, now)
	if f.IntervalDays != 1 || !f.DueAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("first good: %+v", f)
	}
	f = schedule(f, GradeGood, now)
	if f.IntervalDays != 2.5 {
		t.Fatalf("second good: %+v", f)
	}
	f = schedule(f, GradeEasy, now)
	if f.IntervalDays != 8.8 {
		t.Fatalf("easy: %+v", f)
	}
	f = schedule(f, GradeAgain, now)
	if f.IntervalDays != 0 || f.Lapses != 1 || !f.DueAt.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("again: %+v", f)
	}
	if f = schedule(storage.Flashcard{}, GradeEasy, now); f.IntervalDays != 3 {
		t.Fatalf("first easy: %+v", f)
	}
}
