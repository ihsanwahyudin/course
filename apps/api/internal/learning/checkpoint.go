package learning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/assessment"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/progress"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

// Checkpoints are written fresh for every attempt: by the AI from the
// lesson, avoiding the learner's earlier questions, or from the lesson's
// hand-written questions when the AI's don't validate. Answer keys stay on
// the server; the AI grades and explains every answer, but the pass mark is
// applied here.

const (
	checkpointTimeout  = 3 * time.Minute
	earlierCheckpoints = 4 // how many earlier attempts the writer must avoid
	attemptHistory     = 20
)

// CheckItemView is a checkpoint question as the learner sees it.
type CheckItemView struct {
	ID          string   `json:"id"`
	LessonID    string   `json:"lessonId,omitempty"` // level checkpoints: the lesson it comes from
	ObjectiveID string   `json:"objectiveId"`
	Type        string   `json:"type"` // mcq | multi | word_order | letters | match | text
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options,omitempty"` // mcq, multi
	Tokens      []string `json:"tokens,omitempty"`  // word_order: words; letters: letter tiles
	// Left and Right: match, the two columns (Right shuffled).
	Left  []string `json:"left,omitempty"`
	Right []string `json:"right,omitempty"`
	// HasAudio: a listening question, played from …/checkpoints/{id}/items/{itemId}/listen.
	HasAudio bool `json:"hasAudio"`
}

// CheckpointView is an open checkpoint (no answers).
type CheckpointView struct {
	ID       string          `json:"id"`
	Practice bool            `json:"practice"`
	Attempt  int             `json:"attempt"` // 1 for the learner's first attempt at this lesson
	Items    []CheckItemView `json:"items"`
}

func checkpointView(c storage.Checkpoint, attempt int) (*CheckpointView, error) {
	items, err := decodeItems(c)
	if err != nil {
		return nil, err
	}
	v := &CheckpointView{ID: c.ID, Practice: c.Practice, Attempt: attempt, Items: make([]CheckItemView, 0, len(items))}
	for _, it := range items {
		v.Items = append(v.Items, CheckItemView{ID: it.ID, LessonID: it.LessonID, ObjectiveID: it.ObjectiveID, Type: it.Type, Prompt: it.Prompt,
			Options: it.Options, Tokens: it.Tokens, Left: it.Left, Right: it.Right, HasAudio: it.Listen != ""})
	}
	return v, nil
}

func decodeItems(c storage.Checkpoint) ([]tutor.CheckItem, error) {
	var items []tutor.CheckItem
	if err := json.Unmarshal(c.Items, &items); err != nil {
		return nil, fmt.Errorf("checkpoint %s: %w", c.ID, err)
	}
	return items, nil
}

// StartCheckpoint opens the session's checkpoint, writing a new one if none
// is open. After the lesson is mastered it's a practice checkpoint.
func (s *Service) StartCheckpoint(ctx context.Context, userID, sessionID string) (CheckpointView, error) {
	sess, err := s.getOwnSession(ctx, userID, sessionID)
	if err != nil {
		return CheckpointView{}, err
	}
	cur, n, err := s.sessionCourse(sess)
	if err != nil {
		return CheckpointView{}, err
	}
	practice := sess.Phase == string(tutor.PhaseMastered)
	if !practice {
		if sess.EndedAt != nil {
			return CheckpointView{}, errLessonClosed
		}
		// A level checkpoint has no steps of its own.
		if !n.LevelCheck() {
			lc, err := decodeLesson(cur, n, sess)
			if err != nil {
				return CheckpointView{}, err
			}
			if len(lc.plan.Steps) == 0 || sess.StepIndex < len(lc.plan.Steps) {
				return CheckpointView{}, errStepsFirst
			}
		}
	}
	c, err := s.ensureCheckpoint(ctx, userID, sess, practice)
	if err != nil {
		return CheckpointView{}, err
	}
	v, err := s.openCheckpointView(ctx, userID, cur, n, c)
	if err != nil {
		return CheckpointView{}, err
	}
	return *v, nil
}

func (s *Service) openCheckpointView(ctx context.Context, userID string, cur *curriculum.Curriculum, n *curriculum.Node, c storage.Checkpoint) (*CheckpointView, error) {
	done, err := storage.NodeAttempts(ctx, s.Store.Pool, userID, cur.Slug, n.ID, attemptHistory)
	if err != nil {
		return nil, err
	}
	return checkpointView(c, len(done)+1)
}

// ensureCheckpoint returns the session's open checkpoint, writing one if
// needed. One writer per session: concurrent callers wait for it.
func (s *Service) ensureCheckpoint(ctx context.Context, userID string, sess storage.Session, practice bool) (storage.Checkpoint, error) {
	for {
		c, err := storage.OpenCheckpoint(ctx, s.Store.Pool, sess.ID)
		if !errors.Is(err, storage.ErrNotFound) {
			return c, err
		}
		s.cpMu.Lock()
		if s.cpInflight == nil {
			s.cpInflight = map[string]chan struct{}{}
		}
		if wait, ok := s.cpInflight[sess.ID]; ok {
			s.cpMu.Unlock()
			select {
			case <-wait:
				continue
			case <-ctx.Done():
				return storage.Checkpoint{}, errf(504, "checkpoint_slow", true, "The checkpoint is still being prepared. Please try again.")
			}
		}
		done := make(chan struct{})
		s.cpInflight[sess.ID] = done
		s.cpMu.Unlock()
		// Writing continues if the requester goes away; the result is stored.
		wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), checkpointTimeout)
		c, err = s.writeCheckpoint(wctx, userID, sess, practice)
		cancel()
		s.cpMu.Lock()
		delete(s.cpInflight, sess.ID)
		s.cpMu.Unlock()
		close(done)
		return c, err
	}
}

func (s *Service) writeCheckpoint(ctx context.Context, userID string, sess storage.Session, practice bool) (storage.Checkpoint, error) {
	cur, n, err := s.sessionCourse(sess)
	if err != nil {
		return storage.Checkpoint{}, err
	}
	lesson, ok := cur.Lesson(n.ID)
	if !ok && !n.LevelCheck() {
		return storage.Checkpoint{}, errf(409, "no_checkpoint", false, "This lesson has no checkpoint.")
	}
	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return storage.Checkpoint{}, err
	}
	lang := cur.Lang(sess.Locale)
	objs := map[string]bool{}
	for _, o := range n.Objectives {
		objs[o.ID] = true
	}
	var earlier []string
	if past, err := storage.EarlierCheckpoints(ctx, s.Store.Pool, userID, cur.Slug, n.ID, earlierCheckpoints); err == nil {
		for _, raw := range past {
			var items []tutor.CheckItem
			if json.Unmarshal(raw, &items) == nil {
				for _, it := range items {
					earlier = append(earlier, strings.TrimSpace(it.Prompt+" "+it.Listen))
				}
			}
		}
	}
	in := ai.CheckpointInput{LessonInput: s.lessonInput(ctx, st, n, lang), Earlier: earlier}
	// Never the hand-written questions again (they're the fallback), nor earlier ones.
	avoid := append([]string{}, earlier...)
	handWritten := func(l *curriculum.Lesson) {
		for _, it := range l.Assessment {
			avoid = append(avoid, it.Prompt, strings.TrimSpace(l.PromptIn(it, lang)+" "+it.Listen), strings.TrimSpace(it.Prompt+" "+it.Listen))
		}
	}
	validate := func(items []tutor.CheckItem, avoid []string) ([]tutor.CheckItem, error) {
		return tutor.ValidateCheckpoint(items, objs, avoid)
	}
	fallback := func() []tutor.CheckItem { return ai.AuthoredCheckpoint(lesson, lang) }
	if n.LevelCheck() {
		lessons := map[string]string{}
		for _, l := range in.LevelLessons {
			lessons[l.ID] = l.Objective
			if ll, ok := cur.Lesson(l.ID); ok {
				handWritten(ll)
			}
		}
		validate = func(items []tutor.CheckItem, avoid []string) ([]tutor.CheckItem, error) {
			return tutor.ValidateLevelCheckpoint(items, lessons, avoid)
		}
		fallback = func() []tutor.CheckItem { return ai.LevelFallback(in.LevelLessons) }
	} else {
		handWritten(lesson)
		in.Examples = lesson.Assessment
	}
	var items []tutor.CheckItem
	source := "authored"
	for attempt := 0; attempt < 2; attempt++ {
		if err := s.checkQuota(ctx, userID); err != nil {
			break
		}
		res, err := s.AI.Checkpoint(ctx, in)
		s.recordUsage(ctx, userID, "checkpoint", res.Usage)
		if err != nil {
			slog.Warn("checkpoint call failed", "provider", s.AI.Name(), "node", n.ID, "err", err)
			break
		}
		var out struct {
			Items []tutor.DraftItem `json:"items"`
		}
		if err = json.Unmarshal(res.Raw, &out); err == nil {
			items, err = tutor.FromDrafts(out.Items)
		}
		if err == nil {
			err = beginnerTexts(items, n)
		}
		if err == nil {
			if items, err = validate(items, avoid); err == nil {
				// A second look, question by question: drop any with two right
				// answers, a wrong key or the wrong language, then check the rest
				// still covers the lesson.
				kept, problems := s.verifyCheckpoint(ctx, userID, in.LessonInput, items)
				if items, err = validate(kept, avoid); err == nil {
					source = s.AI.Name()
					break
				}
				in.Problems = problems
			}
		}
		if len(in.Problems) == 0 && err != nil {
			in.Problems = []string{err.Error()}
		}
		slog.Warn("checkpoint rejected", "provider", s.AI.Name(), "node", n.ID, "attempt", attempt, "err", err, "problems", in.Problems)
	}
	if source == "authored" {
		if items, err = validate(fallback(), nil); err != nil {
			return storage.Checkpoint{}, fmt.Errorf("hand-written checkpoint for %s: %w", n.ID, err)
		}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return storage.Checkpoint{}, err
	}
	c, err := storage.CreateCheckpoint(ctx, s.Store.Pool, sess.ID, practice, lang, source, raw)
	if errors.Is(err, storage.ErrConflict) {
		return storage.OpenCheckpoint(ctx, s.Store.Pool, sess.ID)
	}
	if err != nil {
		return c, err
	}
	// Listening questions are new sentences: record them now, so they play
	// without a wait.
	var texts []string
	for _, it := range items {
		if it.Listen != "" {
			texts = append(texts, it.Listen)
		}
	}
	s.warmTexts(userID, texts, true)
	return c, nil
}

// beginnerTexts: at A0 and A1 learners mostly tap their answers, so an AI
// checkpoint may hold only a few open questions (ai.MaxTextItems).
func beginnerTexts(items []tutor.CheckItem, n *curriculum.Node) error {
	most, capped := ai.MaxTextItems(n.Level, n.LevelCheck())
	if !capped {
		return nil
	}
	texts := 0
	for _, it := range items {
		if it.Type == tutor.QuestionText {
			texts++
		}
	}
	if texts > most {
		return fmt.Errorf("%d open \"text\" questions; at level %s write at most %d and use mcq, multi, match, letters or word_order for the rest", texts, n.Level, most)
	}
	return nil
}

// verifyCheckpoint asks the AI to check every question and returns the
// ones that passed, with the problems found in the others. If the check
// itself fails, the (already validated) questions are kept.
func (s *Service) verifyCheckpoint(ctx context.Context, userID string, in ai.LessonInput, items []tutor.CheckItem) ([]tutor.CheckItem, []string) {
	if err := s.checkQuota(ctx, userID); err != nil {
		return items, nil
	}
	res, err := s.AI.VerifyCheckpoint(ctx, ai.VerifyInput{LessonInput: in, Items: items})
	s.recordUsage(ctx, userID, "checkpoint", res.Usage)
	var out struct {
		Items []struct {
			ID      string `json:"id"`
			OK      bool   `json:"ok"`
			Problem string `json:"problem"`
		} `json:"items"`
	}
	if err == nil {
		err = json.Unmarshal(res.Raw, &out)
	}
	if err != nil || len(out.Items) == 0 {
		slog.Warn("checkpoint check unavailable; keeping the questions", "err", err)
		return items, nil
	}
	bad := map[string]string{}
	for _, c := range out.Items {
		if !c.OK {
			bad[c.ID] = strings.TrimSpace(c.Problem)
		}
	}
	var kept []tutor.CheckItem
	var problems []string
	for _, it := range items {
		if p, isBad := bad[it.ID]; isBad {
			problems = append(problems, fmt.Sprintf("%q — %s", it.Prompt, p))
			continue
		}
		kept = append(kept, it)
	}
	return kept, problems
}

// prefetchCheckpoint writes the next checkpoint in the background, so the
// learner doesn't wait when they get there.
func (s *Service) prefetchCheckpoint(userID string, sess storage.Session) {
	if !s.PrefetchCheckpoints {
		return
	}
	s.warm.Add(1)
	go func() {
		defer s.warm.Done()
		ctx, cancel := context.WithTimeout(context.Background(), checkpointTimeout)
		defer cancel()
		if _, err := s.ensureCheckpoint(ctx, userID, sess, false); err != nil {
			slog.Warn("prefetch checkpoint", "session", sess.ID, "err", err)
		}
	}()
}

// CheckpointListen plays a checkpoint's listening question (recorded live:
// these sentences are new for every attempt).
func (s *Service) CheckpointListen(ctx context.Context, userID, sessionID, checkpointID, itemID string) (ListenAudio, error) {
	sess, err := s.getOwnSession(ctx, userID, sessionID)
	if err != nil {
		return ListenAudio{}, err
	}
	c, err := storage.GetCheckpoint(ctx, s.Store.Pool, sess.ID, checkpointID)
	if errors.Is(err, storage.ErrNotFound) {
		return ListenAudio{}, errf(404, "item_not_found", false, "No such checkpoint question.")
	}
	if err != nil {
		return ListenAudio{}, err
	}
	items, err := decodeItems(c)
	if err != nil {
		return ListenAudio{}, err
	}
	for _, it := range items {
		if it.ID == itemID {
			return s.listenAudio(ctx, userID, it.Listen, true)
		}
	}
	return ListenAudio{}, errf(404, "item_not_found", false, "No such checkpoint question.")
}

// SubmitAssessment grades the session's open checkpoint.
func (s *Service) SubmitAssessment(ctx context.Context, userID, sessionID string, answers map[string]string) (AssessmentResult, error) {
	return s.SubmitCheckpoint(ctx, userID, sessionID, "", answers)
}

// SubmitCheckpoint grades a checkpoint (the open one when checkpointID is
// ""): fixed verdicts first, then the AI judges and explains every answer,
// and the pass mark decides mastery. Practice attempts are only recorded.
func (s *Service) SubmitCheckpoint(ctx context.Context, userID, sessionID, checkpointID string, answers map[string]string) (AssessmentResult, error) {
	sess, err := s.getOwnSession(ctx, userID, sessionID)
	if err != nil {
		return AssessmentResult{}, err
	}
	cur, n, err := s.sessionCourse(sess)
	if err != nil {
		return AssessmentResult{}, err
	}
	var c storage.Checkpoint
	if checkpointID != "" {
		c, err = storage.GetCheckpoint(ctx, s.Store.Pool, sess.ID, checkpointID)
	} else {
		c, err = storage.OpenCheckpoint(ctx, s.Store.Pool, sess.ID)
	}
	// A submission racing another one for the same checkpoint may find it
	// already submitted: read the session again before calling it an error.
	masteredNow := func() bool {
		fresh, ferr := storage.GetSession(ctx, s.Store.Pool, sess.ID, userID)
		return ferr == nil && fresh.Phase == string(tutor.PhaseMastered)
	}
	switch {
	case errors.Is(err, storage.ErrNotFound) && masteredNow():
		return s.completedResult(ctx, userID, sess)
	case errors.Is(err, storage.ErrNotFound):
		return AssessmentResult{}, errf(409, "no_checkpoint", false, "Start the checkpoint first.")
	case err != nil:
		return AssessmentResult{}, err
	case c.SubmittedAt != nil && masteredNow():
		return s.completedResult(ctx, userID, sess)
	case c.SubmittedAt != nil:
		return AssessmentResult{}, errf(409, "checkpoint_submitted", false, "These answers were already sent. Start the checkpoint again for new questions.")
	}
	if !c.Practice && !n.LevelCheck() {
		if lc, err := decodeLesson(cur, n, sess); err != nil {
			return AssessmentResult{}, err
		} else if len(lc.plan.Steps) == 0 || sess.StepIndex < len(lc.plan.Steps) {
			return AssessmentResult{}, errStepsFirst
		}
	}
	items, err := decodeItems(c)
	if err != nil {
		return AssessmentResult{}, err
	}
	clean, err := checkAnswers(items, answers)
	if err != nil {
		return AssessmentResult{}, err
	}

	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return AssessmentResult{}, err
	}
	lang := cur.Lang(c.Locale)
	fixed := map[string]int{}
	for _, it := range items {
		if score, ok := tutor.FixedVerdict(it, clean[it.ID], ai.StrictMatch); ok {
			fixed[it.ID] = score
		}
	}
	review, scores, feedback, raw, err := s.gradeCheckpoint(ctx, userID, st, n, lang, items, clean, fixed)
	if err != nil {
		return AssessmentResult{}, err
	}
	scored, _ := json.Marshal(map[string]any{"objectiveScores": scores, "evidence": review.Strengths, "feedback": feedback})
	res, err := assessment.Score(scored, n)
	if err != nil {
		slog.Warn("checkpoint result rejected", "session", sess.ID, "err", err)
		return AssessmentResult{}, errf(502, "evaluator_malformed", true, "Scoring returned an invalid result. Nothing was saved — please submit again.")
	}
	res.Evidence = review.Strengths

	var out AssessmentResult
	err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		locked, err := storage.LockSession(ctx, tx, sess.ID, userID)
		if err != nil {
			return err
		}
		lc, err := storage.LockCheckpoint(ctx, tx, sess.ID, c.ID)
		if err != nil {
			return err
		}
		if lc.SubmittedAt != nil {
			if locked.Phase == string(tutor.PhaseMastered) {
				out.AlreadyCompleted = true
				return nil
			}
			return errf(409, "checkpoint_submitted", false, "These answers were already sent. Start the checkpoint again for new questions.")
		}
		if !c.Practice && locked.Phase == string(tutor.PhaseMastered) {
			out.AlreadyCompleted = true
			return nil
		}
		before, err := s.load(ctx, tx, userID, cur)
		if err != nil {
			return err
		}
		if !c.Practice {
			if _, err := storage.LockProgress(ctx, tx, userID, cur.Slug, n.ID); err != nil {
				return err
			}
		}
		out.Attempt, err = storage.InsertAttempt(ctx, tx, storage.NewAttempt{
			SessionID: sess.ID, CheckpointID: c.ID, Practice: c.Practice, Score: res.Score, Threshold: res.Threshold, Passed: res.Passed,
			ObjectiveScores: res.ObjectiveScores, Evidence: res.Evidence, Feedback: res.Feedback, Review: review,
			Answers: clean, EvaluatorResponse: raw, Evaluator: s.AI.Name(),
		})
		if err != nil {
			return err
		}
		if err := storage.SubmitCheckpoint(ctx, tx, c.ID); err != nil {
			return err
		}
		out.Attempt.CourseSlug, out.Attempt.NodeID = cur.Slug, n.ID
		out.Passed = res.Passed
		if c.Practice {
			return nil
		}
		if res.Passed {
			if out.NewlyMastered, err = storage.MarkMastered(ctx, tx, userID, cur.Slug, n.ID, res.Score); err != nil {
				return err
			}
			if err := storage.EndSession(ctx, tx, sess.ID, string(tutor.PhaseMastered)); err != nil {
				return err
			}
		}
		after, err := s.load(ctx, tx, userID, cur)
		if err != nil {
			return err
		}
		for _, id := range progress.NewlyUnlocked(cur, before.statuses, after.statuses) {
			un, _ := cur.Node(id)
			out.Unlocked = append(out.Unlocked, nodeView(after, un))
		}
		if err := storage.SetEnrollmentLevel(ctx, tx, userID, cur.Slug, progress.CurrentLevel(cur, after.statuses)); err != nil {
			return err
		}
		return storage.TouchEnrollment(ctx, tx, userID, cur.Slug)
	})
	if err != nil {
		return AssessmentResult{}, err
	}
	if out.AlreadyCompleted {
		return s.completedResult(ctx, userID, sess)
	}
	if out.Unlocked == nil {
		out.Unlocked = []NodeView{}
	}
	s.saveCheckpointMistakes(ctx, userID, cur, n, review)
	if out.NewlyMastered && !n.LevelCheck() {
		s.finishLesson(ctx, userID, cur, n, sess, res.ObjectiveScores)
	}
	if !res.Passed && !c.Practice {
		s.prefetchCheckpoint(userID, sess)
	}
	out.Session, err = s.GetSession(ctx, userID, sess.ID)
	return out, err
}

// checkAnswers trims the answers and requires one for every question.
func checkAnswers(items []tutor.CheckItem, answers map[string]string) (map[string]string, error) {
	known := map[string]bool{}
	clean := map[string]string{}
	for _, it := range items {
		known[it.ID] = true
		a := strings.TrimSpace(answers[it.ID])
		if a == "" {
			return nil, errf(400, "invalid_answers", false, "Answer every question (missing %s).", it.ID)
		}
		if utf8.RuneCountInString(a) > assessment.MaxAnswerRunes {
			return nil, errf(400, "invalid_answers", false, "Answer %s is too long.", it.ID)
		}
		if !tutor.AnswerFits(it, a) {
			return nil, errf(400, "invalid_answers", false, "Answer %s doesn't fit its question.", it.ID)
		}
		clean[it.ID] = a
	}
	for id := range answers {
		if !known[id] {
			return nil, errf(400, "invalid_answers", false, "Unknown question %q.", id)
		}
	}
	return clean, nil
}

// gradeCheckpoint asks the AI to judge and explain every answer (one retry).
// Nothing is saved when it fails; the learner submits again.
func (s *Service) gradeCheckpoint(ctx context.Context, userID string, st state, n *curriculum.Node, lang string, items []tutor.CheckItem,
	answers map[string]string, fixed map[string]int) (tutor.Review, map[string]int, string, json.RawMessage, error) {
	cur := st.cur
	in := ai.GradeInput{
		LessonInput: s.lessonInput(ctx, st, n, lang), EvaluatorPolicy: cur.Tutor.EvaluatorPolicy,
		Items: items, Answers: answers, Fixed: fixed, GoalHint: goalHint(cur, st.profile),
	}
	for attempt := 0; attempt < 2; attempt++ {
		provider := s.AI
		if err := s.checkQuota(ctx, userID); errors.Is(err, errQuota) {
			// Over the day's AI budget: grade against the answer key offline
			// rather than lock the learner out of the checkpoint.
			provider = &ai.Mock{}
		} else if err != nil {
			return tutor.Review{}, nil, "", nil, err
		}
		res, err := provider.GradeCheckpoint(ctx, in)
		s.recordUsage(ctx, userID, "evaluate", res.Usage)
		if err != nil {
			slog.Warn("checkpoint grading failed", "provider", s.AI.Name(), "err", err)
			return tutor.Review{}, nil, "", nil, errf(502, "evaluator_unavailable", true, "Your answers couldn't be scored right now. Nothing was saved — please submit again.")
		}
		review, scores, feedback, err := tutor.ApplyGrade(res.Raw, items, answers, fixed)
		if err == nil {
			return review, scores, feedback, res.Raw, nil
		}
		slog.Warn("checkpoint grade rejected", "provider", s.AI.Name(), "attempt", attempt, "err", err)
	}
	return tutor.Review{}, nil, "", nil, errf(502, "evaluator_malformed", true, "Scoring returned an invalid result. Nothing was saved — please submit again.")
}

// saveCheckpointMistakes turns wrong checkpoint answers into review flashcards.
func (s *Service) saveCheckpointMistakes(ctx context.Context, userID string, cur *curriculum.Curriculum, n *curriculum.Node, r tutor.Review) {
	var cards []storage.NewFlashcard
	for _, it := range r.Items {
		if it.Verdict == tutor.VerdictCorrect {
			continue
		}
		front := it.Prompt
		if it.Listen != "" {
			front += " — " + it.Listen
		}
		if utf8.RuneCountInString(front) > curriculum.FlashcardMax {
			front = string([]rune(front)[:curriculum.FlashcardMax-1]) + "…"
		}
		back := it.CorrectAnswer
		if utf8.RuneCountInString(back) > curriculum.FlashcardMax {
			back = string([]rune(back)[:curriculum.FlashcardMax-1]) + "…"
		}
		cards = append(cards, storage.NewFlashcard{Front: front, Back: back})
	}
	if len(cards) == 0 {
		return
	}
	if total, _, err := storage.CountFlashcards(ctx, s.Store.Pool, userID); err != nil || total+len(cards) > maxCardsPerUser {
		return
	}
	if _, err := storage.InsertFlashcards(ctx, s.Store.Pool, userID, cur.Slug, n.ID, "mistake", cards); err != nil {
		slog.Warn("save checkpoint flashcards", "err", err)
	}
}

// AttemptSummary is one entry of a lesson's checkpoint history.
type AttemptSummary struct {
	ID        string    `json:"id"`
	Score     int       `json:"score"`
	Threshold int       `json:"threshold"`
	Passed    bool      `json:"passed"`
	Practice  bool      `json:"practice"`
	CreatedAt time.Time `json:"createdAt"`
}

func attemptSummaries(as []storage.Attempt) []AttemptSummary {
	out := make([]AttemptSummary, 0, len(as))
	for _, a := range as {
		out = append(out, AttemptSummary{a.ID, a.Score, a.Threshold, a.Passed, a.Practice, a.CreatedAt})
	}
	return out
}

// Attempt returns one of the learner's checkpoint attempts with its review.
func (s *Service) Attempt(ctx context.Context, userID, id string) (storage.Attempt, error) {
	a, err := storage.GetAttempt(ctx, s.Store.Pool, userID, id)
	if errors.Is(err, storage.ErrNotFound) {
		return a, errf(404, "attempt_not_found", false, "Attempt not found.")
	}
	return a, err
}
