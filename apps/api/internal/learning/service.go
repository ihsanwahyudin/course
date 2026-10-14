// Package learning orchestrates the learning loop: roadmap views, lesson
// sessions, tutor turns, assessments and mastery. It is the only place that
// changes progress, and it does so from validated scores and fixed rules.
package learning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/assessment"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/progress"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

const MaxMessageRunes = 2000

// Error is a client-facing failure with a stable code.
type Error struct {
	Status    int    `json:"-"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func errf(status int, code string, retryable bool, format string, args ...any) *Error {
	return &Error{Status: status, Code: code, Message: fmt.Sprintf(format, args...), Retryable: retryable}
}

var (
	errAIUnavailable = errf(502, "tutor_unavailable", true, "The tutor is not responding right now. Please try again.")
	errBusy          = errf(409, "session_busy", true, "This lesson was updated in another tab. Please retry.")
)

type Service struct {
	Store     *storage.Store
	Cur       *curriculum.Curriculum
	AI        ai.Provider
	LearnerID string
}

// ---- views ----

type NodeView struct {
	*curriculum.Node
	Status       progress.Status `json:"status"`
	MasteryScore *int            `json:"masteryScore"`
	SuccessorIDs []string        `json:"successorIds"`
}

type LevelSummary struct {
	Level       string `json:"level"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Total       int    `json:"total"`
	Ready       int    `json:"ready"`
	Mastered    int    `json:"mastered"`
}

type Summary struct {
	TotalNodes     int            `json:"totalNodes"`
	ReadyNodes     int            `json:"readyNodes"`
	MasteredNodes  int            `json:"masteredNodes"`
	TargetNodes    int            `json:"targetNodes"` // nodes up to and including the target level
	TargetMastered int            `json:"targetMastered"`
	Levels         []LevelSummary `json:"levels"`
}

type RoadmapView struct {
	Version    string          `json:"version"`
	Learner    storage.Learner `json:"learner"`
	Nodes      []NodeView      `json:"nodes"`
	NextNodeID *string         `json:"nextNodeId"`
	Summary    Summary         `json:"summary"`
}

type state struct {
	learner  storage.Learner
	stored   map[string]progress.Stored
	statuses map[string]progress.Status
}

func (s *Service) load(ctx context.Context, db storage.DB) (state, error) {
	l, err := storage.GetLearner(ctx, db, s.LearnerID)
	if err != nil {
		return state{}, err
	}
	rows, err := storage.ListProgress(ctx, db, s.LearnerID)
	if err != nil {
		return state{}, err
	}
	stored := map[string]progress.Stored{}
	for _, r := range rows {
		stored[r.NodeID] = progress.Stored{Status: r.Status, MasteryScore: r.MasteryScore}
	}
	return state{learner: l, stored: stored, statuses: progress.Derive(s.Cur, stored)}, nil
}

func (s *Service) nodeView(n *curriculum.Node, st state) NodeView {
	succ := s.Cur.Successors(n.ID)
	if succ == nil {
		succ = []string{}
	}
	return NodeView{Node: n, Status: st.statuses[n.ID], MasteryScore: st.stored[n.ID].MasteryScore, SuccessorIDs: succ}
}

func (s *Service) summary(st state) Summary {
	sum := Summary{}
	target := curriculum.LevelIndex(st.learner.TargetLevel)
	byLevel := map[string]*LevelSummary{}
	for _, li := range s.Cur.Levels {
		ls := &LevelSummary{Level: li.Level, Title: li.Title, Description: li.Description}
		byLevel[li.Level] = ls
	}
	for _, n := range s.Cur.Nodes {
		ls := byLevel[n.Level]
		mastered := st.statuses[n.ID] == progress.StatusMastered
		ls.Total++
		sum.TotalNodes++
		if n.Ready() {
			ls.Ready++
			sum.ReadyNodes++
		}
		if mastered {
			ls.Mastered++
			sum.MasteredNodes++
		}
		if curriculum.LevelIndex(n.Level) <= target {
			sum.TargetNodes++
			if mastered {
				sum.TargetMastered++
			}
		}
	}
	for _, li := range s.Cur.Levels {
		sum.Levels = append(sum.Levels, *byLevel[li.Level])
	}
	return sum
}

func (s *Service) Roadmap(ctx context.Context) (RoadmapView, error) {
	st, err := s.load(ctx, s.Store.Pool)
	if err != nil {
		return RoadmapView{}, err
	}
	v := RoadmapView{Version: s.Cur.Version, Learner: st.learner, Summary: s.summary(st)}
	for _, n := range s.Cur.Nodes {
		v.Nodes = append(v.Nodes, s.nodeView(n, st))
	}
	if next := progress.NextSuggested(s.Cur, st.statuses); next != nil {
		v.NextNodeID = &next.ID
	}
	return v, nil
}

type NodeDetail struct {
	NodeView
	Prerequisites []NodeView `json:"prerequisites"`
	Successors    []NodeView `json:"successors"`
	CanStart      bool       `json:"canStart"`
	BlockedReason string     `json:"blockedReason,omitempty"`
	OpenSessionID *string    `json:"openSessionId"`
	KeyPoints     []string   `json:"keyPoints"`
}

func (s *Service) Node(ctx context.Context, id string) (NodeDetail, error) {
	n, ok := s.Cur.Node(id)
	if !ok {
		return NodeDetail{}, errf(404, "node_not_found", false, "No roadmap node %q.", id)
	}
	st, err := s.load(ctx, s.Store.Pool)
	if err != nil {
		return NodeDetail{}, err
	}
	d := NodeDetail{NodeView: s.nodeView(n, st), Prerequisites: []NodeView{}, Successors: []NodeView{}, KeyPoints: []string{}}
	for _, p := range n.PrerequisiteIDs {
		pn, _ := s.Cur.Node(p)
		d.Prerequisites = append(d.Prerequisites, s.nodeView(pn, st))
	}
	for _, c := range s.Cur.Successors(n.ID) {
		cn, _ := s.Cur.Node(c)
		d.Successors = append(d.Successors, s.nodeView(cn, st))
	}
	d.CanStart, d.BlockedReason = s.startable(n, st)
	if l, ok := s.Cur.Lesson(n.ID); ok {
		for _, k := range l.KeyPoints {
			d.KeyPoints = append(d.KeyPoints, k.Title)
		}
	}
	if sess, err := storage.FindOpenSession(ctx, s.Store.Pool, s.LearnerID, n.ID); err == nil {
		d.OpenSessionID = &sess.ID
	} else if !errors.Is(err, storage.ErrNotFound) {
		return NodeDetail{}, err
	}
	return d, nil
}

func (s *Service) startable(n *curriculum.Node, st state) (bool, string) {
	switch st.statuses[n.ID] {
	case progress.StatusComingSoon:
		return false, "This lesson is coming soon."
	case progress.StatusLocked:
		var missing []string
		for _, p := range n.PrerequisiteIDs {
			if st.statuses[p] != progress.StatusMastered {
				pn, _ := s.Cur.Node(p)
				missing = append(missing, pn.Title)
			}
		}
		return false, "Master first: " + strings.Join(missing, ", ") + "."
	}
	return true, ""
}

// ---- sessions ----

type AssessmentItemView struct {
	ID          string `json:"id"`
	ObjectiveID string `json:"objectiveId"`
	Prompt      string `json:"prompt"`
}

type SessionView struct {
	storage.Session
	Node        NodeView             `json:"node"`
	Messages    []storage.Message    `json:"messages"`
	Steps       []tutor.Phase        `json:"steps"`
	CanChat     bool                 `json:"canChat"`
	Assessment  []AssessmentItemView `json:"assessment"`
	LastAttempt *storage.Attempt     `json:"lastAttempt"`
}

func (s *Service) sessionView(ctx context.Context, db storage.DB, sess storage.Session) (SessionView, error) {
	st, err := s.load(ctx, db)
	if err != nil {
		return SessionView{}, err
	}
	n, ok := s.Cur.Node(sess.NodeID)
	if !ok {
		return SessionView{}, errf(410, "node_removed", false, "This lesson is no longer in the curriculum.")
	}
	msgs, err := storage.ListMessages(ctx, db, sess.ID)
	if err != nil {
		return SessionView{}, err
	}
	v := SessionView{
		Session: sess, Node: s.nodeView(n, st), Messages: msgs, Steps: tutor.Steps,
		CanChat:    sess.EndedAt == nil && tutor.Phase(sess.Phase).AcceptsChat(),
		Assessment: []AssessmentItemView{},
	}
	if sess.Phase == string(tutor.PhaseAssessment) {
		if l, ok := s.Cur.Lesson(n.ID); ok {
			for _, it := range l.Assessment {
				v.Assessment = append(v.Assessment, AssessmentItemView{it.ID, it.ObjectiveID, it.Prompt})
			}
		}
	}
	if a, err := storage.LatestAttempt(ctx, db, sess.ID); err == nil {
		v.LastAttempt = &a
	} else if !errors.Is(err, storage.ErrNotFound) {
		return SessionView{}, err
	}
	return v, nil
}

// StartSession opens (or resumes) the learner's session for a ready node.
func (s *Service) StartSession(ctx context.Context, nodeID string) (SessionView, error) {
	n, ok := s.Cur.Node(nodeID)
	if !ok {
		return SessionView{}, errf(404, "node_not_found", false, "No roadmap node %q.", nodeID)
	}
	st, err := s.load(ctx, s.Store.Pool)
	if err != nil {
		return SessionView{}, err
	}
	if ok, reason := s.startable(n, st); !ok {
		code := "node_locked"
		if st.statuses[n.ID] == progress.StatusComingSoon {
			code = "node_coming_soon"
		}
		return SessionView{}, errf(409, code, false, "%s", reason)
	}
	lesson, _ := s.Cur.Lesson(n.ID)

	sess, err := storage.FindOpenSession(ctx, s.Store.Pool, s.LearnerID, n.ID)
	if errors.Is(err, storage.ErrNotFound) {
		err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
			var err error
			sess, err = storage.CreateSession(ctx, tx, s.LearnerID, n.ID, string(tutor.PhaseIntroduction))
			if err != nil {
				return err
			}
			if _, err := storage.AddMessage(ctx, tx, sess.ID, "assistant", lesson.Intro, sess.Phase, map[string]any{"kind": "intro"}); err != nil {
				return err
			}
			return storage.MarkInProgress(ctx, tx, s.LearnerID, n.ID)
		})
		if errors.Is(err, storage.ErrConflict) {
			sess, err = storage.FindOpenSession(ctx, s.Store.Pool, s.LearnerID, n.ID)
		}
	}
	if err != nil {
		return SessionView{}, err
	}
	return s.sessionView(ctx, s.Store.Pool, sess)
}

func (s *Service) GetSession(ctx context.Context, id string) (SessionView, error) {
	sess, err := storage.GetSession(ctx, s.Store.Pool, id, s.LearnerID)
	if errors.Is(err, storage.ErrNotFound) {
		return SessionView{}, errf(404, "session_not_found", false, "Session not found.")
	}
	if err != nil {
		return SessionView{}, err
	}
	return s.sessionView(ctx, s.Store.Pool, sess)
}

const assessmentIntro = "Time for your assessment. Answer each question on your own — there are no hints here. " +
	"Your answers are scored against this lesson's objectives, and you need %d/100 to master it."

// SendMessage records a learner reply, asks the tutor for a response and
// applies the phase rules. Nothing is stored unless the tutor reply is valid,
// so a failed call can simply be retried with the same text.
func (s *Service) SendMessage(ctx context.Context, sessionID, content string) (SessionView, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return SessionView{}, errf(400, "empty_message", false, "Message cannot be empty.")
	}
	if utf8.RuneCountInString(content) > MaxMessageRunes {
		return SessionView{}, errf(400, "message_too_long", false, "Message is too long (max %d characters).", MaxMessageRunes)
	}
	sess, err := storage.GetSession(ctx, s.Store.Pool, sessionID, s.LearnerID)
	if errors.Is(err, storage.ErrNotFound) {
		return SessionView{}, errf(404, "session_not_found", false, "Session not found.")
	}
	if err != nil {
		return SessionView{}, err
	}
	phase := tutor.Phase(sess.Phase)
	if sess.EndedAt != nil || !phase.AcceptsChat() {
		return SessionView{}, errf(409, "wrong_phase", false, "This lesson is in %s and doesn't accept chat messages.", phase)
	}
	n, _ := s.Cur.Node(sess.NodeID)
	lesson, _ := s.Cur.Lesson(sess.NodeID)

	st, err := s.load(ctx, s.Store.Pool)
	if err != nil {
		return SessionView{}, err
	}
	history, err := storage.ListMessages(ctx, s.Store.Pool, sess.ID)
	if err != nil {
		return SessionView{}, err
	}
	in := ai.TutorInput{
		CurrentLevel: st.learner.CurrentLevel, TargetLevel: st.learner.TargetLevel,
		Node: n, Lesson: lesson, Phase: phase, NextPhase: tutor.NextOf(phase),
		TurnsInPhase: sess.PhaseTurns + 1,
	}
	for _, m := range history {
		if m.Role == "user" || m.Role == "assistant" {
			in.Messages = append(in.Messages, ai.ChatMessage{Role: m.Role, Content: m.Content})
		}
	}
	in.Messages = append(in.Messages, ai.ChatMessage{Role: "user", Content: content})
	for _, cn := range s.Cur.Nodes {
		if st.statuses[cn.ID] == progress.StatusMastered {
			in.MasteredNodeTitles = append(in.MasteredNodeTitles, cn.Title)
		}
	}
	in.ObservedWeaknesses, err = s.weaknesses(ctx, sess)
	if err != nil {
		return SessionView{}, err
	}

	raw, err := s.AI.TutorReply(ctx, in)
	if err != nil {
		slog.Warn("tutor call failed", "provider", s.AI.Name(), "session", sess.ID, "err", err)
		return SessionView{}, errAIUnavailable
	}
	reply, err := tutor.ParseReply(raw)
	if err != nil {
		slog.Warn("tutor reply rejected", "provider", s.AI.Name(), "session", sess.ID, "err", err)
		return SessionView{}, errAIUnavailable
	}

	turns := sess.PhaseTurns + 1
	next, advanced := tutor.Transition(phase, turns, reply.Action)
	nextTurns := turns
	if advanced {
		nextTurns = 0
	}
	err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		if err := storage.AdvanceSession(ctx, tx, sess.ID, sess.Phase, sess.PhaseTurns, string(next), nextTurns); err != nil {
			return err
		}
		if _, err := storage.AddMessage(ctx, tx, sess.ID, "user", content, sess.Phase, nil); err != nil {
			return err
		}
		meta := map[string]any{"action": reply.Action, "observedMistakes": reply.ObservedMistakes}
		if advanced {
			meta["advancedTo"] = next
		}
		if _, err := storage.AddMessage(ctx, tx, sess.ID, "assistant", reply.Message, sess.Phase, meta); err != nil {
			return err
		}
		if next == tutor.PhaseAssessment {
			msg := fmt.Sprintf(assessmentIntro, n.MasteryThreshold)
			if _, err := storage.AddMessage(ctx, tx, sess.ID, "assistant", msg, string(next), map[string]any{"kind": "assessment_intro"}); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, storage.ErrConflict) {
		return SessionView{}, errBusy
	}
	if err != nil {
		return SessionView{}, err
	}
	return s.GetSession(ctx, sess.ID)
}

// weaknesses combines recent tutor-observed mistake tags with objectives that
// scored low on this session's last attempt.
func (s *Service) weaknesses(ctx context.Context, sess storage.Session) ([]string, error) {
	tags, err := storage.RecentMistakes(ctx, s.Store.Pool, s.LearnerID, 5)
	if err != nil {
		return nil, err
	}
	if a, err := storage.LatestAttempt(ctx, s.Store.Pool, sess.ID); err == nil && !a.Passed {
		n, _ := s.Cur.Node(sess.NodeID)
		for _, o := range n.Objectives {
			if a.ObjectiveScores[o.ID] < a.Threshold {
				tags = append(tags, fmt.Sprintf("objective %q scored %d", o.Text, a.ObjectiveScores[o.ID]))
			}
		}
	} else if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	return tags, nil
}

// ---- assessment ----

type AssessmentResult struct {
	Attempt          storage.Attempt `json:"attempt"`
	Passed           bool            `json:"passed"`
	NewlyMastered    bool            `json:"newlyMastered"`
	AlreadyCompleted bool            `json:"alreadyCompleted"`
	Unlocked         []NodeView      `json:"unlocked"`
	Session          SessionView     `json:"session"`
}

func (s *Service) SubmitAssessment(ctx context.Context, sessionID string, answers map[string]string) (AssessmentResult, error) {
	sess, err := storage.GetSession(ctx, s.Store.Pool, sessionID, s.LearnerID)
	if errors.Is(err, storage.ErrNotFound) {
		return AssessmentResult{}, errf(404, "session_not_found", false, "Session not found.")
	}
	if err != nil {
		return AssessmentResult{}, err
	}
	if sess.Phase == string(tutor.PhaseMastered) {
		return s.completedResult(ctx, sess)
	}
	if sess.Phase != string(tutor.PhaseAssessment) {
		return AssessmentResult{}, errf(409, "wrong_phase", false, "This lesson isn't ready for assessment yet (current phase: %s).", sess.Phase)
	}
	n, _ := s.Cur.Node(sess.NodeID)
	lesson, _ := s.Cur.Lesson(sess.NodeID)
	clean := map[string]string{}
	for k, v := range answers {
		clean[k] = strings.TrimSpace(v)
	}
	if err := assessment.ValidateAnswers(lesson.Assessment, clean); err != nil {
		return AssessmentResult{}, errf(400, "invalid_answers", false, "%s", err.Error())
	}

	st, err := s.load(ctx, s.Store.Pool)
	if err != nil {
		return AssessmentResult{}, err
	}
	raw, err := s.AI.Evaluate(ctx, ai.EvalInput{TargetLevel: st.learner.TargetLevel, Node: n, Items: lesson.Assessment, Answers: clean})
	if err != nil {
		slog.Warn("evaluator call failed", "provider", s.AI.Name(), "session", sess.ID, "err", err)
		return AssessmentResult{}, errf(502, "evaluator_unavailable", true, "Your answers couldn't be scored right now. Nothing was saved — please submit again.")
	}
	res, err := assessment.Score(raw, n)
	if err != nil {
		slog.Warn("evaluator result rejected", "provider", s.AI.Name(), "session", sess.ID, "err", err)
		return AssessmentResult{}, errf(502, "evaluator_malformed", true, "Scoring returned an invalid result. Nothing was saved — please submit again.")
	}

	var out AssessmentResult
	err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		locked, err := storage.LockSession(ctx, tx, sess.ID, s.LearnerID)
		if err != nil {
			return err
		}
		if locked.Phase == string(tutor.PhaseMastered) {
			out.AlreadyCompleted = true
			return nil
		}
		if locked.Phase != string(tutor.PhaseAssessment) {
			return errf(409, "wrong_phase", false, "This assessment was already submitted.")
		}
		before, err := s.load(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := storage.LockProgress(ctx, tx, s.LearnerID, n.ID); err != nil {
			return err
		}
		out.Attempt, err = storage.InsertAttempt(ctx, tx, storage.NewAttempt{
			SessionID: sess.ID, Score: res.Score, Threshold: res.Threshold, Passed: res.Passed,
			ObjectiveScores: res.ObjectiveScores, Evidence: res.Evidence, Feedback: res.Feedback,
			Answers: clean, EvaluatorResponse: json.RawMessage(raw), Evaluator: s.AI.Name(),
		})
		if err != nil {
			return err
		}
		out.Passed = res.Passed
		if res.Passed {
			if out.NewlyMastered, err = storage.MarkMastered(ctx, tx, s.LearnerID, n.ID, res.Score); err != nil {
				return err
			}
			if err := storage.EndSession(ctx, tx, sess.ID, string(tutor.PhaseMastered)); err != nil {
				return err
			}
			msg := fmt.Sprintf("You scored %d/100 and mastered %q. %s", res.Score, n.Title, res.Feedback)
			if _, err := storage.AddMessage(ctx, tx, sess.ID, "assistant", msg, string(tutor.PhaseMastered), map[string]any{"kind": "result"}); err != nil {
				return err
			}
		} else {
			if err := storage.SetSessionPhase(ctx, tx, sess.ID, string(tutor.PhaseReview)); err != nil {
				return err
			}
			var weak []string
			for _, o := range n.Objectives {
				if res.ObjectiveScores[o.ID] < res.Threshold {
					weak = append(weak, strings.ToLower(o.Text[:1])+o.Text[1:])
				}
			}
			msg := fmt.Sprintf("You scored %d/100 — you need %d to master this lesson. %s Let's review: %s. Send a message when you're ready to practise.",
				res.Score, res.Threshold, res.Feedback, strings.Join(weak, "; "))
			if _, err := storage.AddMessage(ctx, tx, sess.ID, "assistant", msg, string(tutor.PhaseReview), map[string]any{"kind": "review_intro"}); err != nil {
				return err
			}
		}
		after, err := s.load(ctx, tx)
		if err != nil {
			return err
		}
		for _, id := range progress.NewlyUnlocked(s.Cur, before.statuses, after.statuses) {
			un, _ := s.Cur.Node(id)
			out.Unlocked = append(out.Unlocked, s.nodeView(un, after))
		}
		return storage.SetLearnerLevel(ctx, tx, s.LearnerID, progress.CurrentLevel(s.Cur, after.statuses))
	})
	if err != nil {
		return AssessmentResult{}, err
	}
	if out.AlreadyCompleted {
		return s.completedResult(ctx, sess)
	}
	if out.Unlocked == nil {
		out.Unlocked = []NodeView{}
	}
	out.Session, err = s.GetSession(ctx, sess.ID)
	return out, err
}

// completedResult answers a repeated submission for an already-mastered
// session without re-scoring or re-awarding anything.
func (s *Service) completedResult(ctx context.Context, sess storage.Session) (AssessmentResult, error) {
	view, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		return AssessmentResult{}, err
	}
	out := AssessmentResult{AlreadyCompleted: true, Passed: true, Unlocked: []NodeView{}, Session: view}
	if view.LastAttempt != nil {
		out.Attempt = *view.LastAttempt
	}
	return out, nil
}

// ---- progress ----

type MasteredItem struct {
	NodeID     string `json:"nodeId"`
	Title      string `json:"title"`
	Level      string `json:"level"`
	Score      *int   `json:"score"`
	MasteredAt string `json:"masteredAt"`
}

type AttemptItem struct {
	storage.Attempt
	Title string `json:"title"`
}

type OpenSessionItem struct {
	SessionID string `json:"sessionId"`
	NodeID    string `json:"nodeId"`
	Title     string `json:"title"`
	Phase     string `json:"phase"`
}

type Weekly struct {
	Messages int `json:"messages"`
	Attempts int `json:"attempts"`
	Mastered int `json:"mastered"`
}

type ProgressView struct {
	Learner      storage.Learner   `json:"learner"`
	Summary      Summary           `json:"summary"`
	NextNode     *NodeView         `json:"nextNode"`
	Mastered     []MasteredItem    `json:"mastered"`
	Attempts     []AttemptItem     `json:"recentAttempts"`
	OpenSessions []OpenSessionItem `json:"openSessions"`
	Weekly       Weekly            `json:"weekly"`
}

func (s *Service) Progress(ctx context.Context) (ProgressView, error) {
	db := s.Store.Pool
	st, err := s.load(ctx, db)
	if err != nil {
		return ProgressView{}, err
	}
	v := ProgressView{Learner: st.learner, Summary: s.summary(st), Mastered: []MasteredItem{}, Attempts: []AttemptItem{}, OpenSessions: []OpenSessionItem{}}
	if next := progress.NextSuggested(s.Cur, st.statuses); next != nil {
		nv := s.nodeView(next, st)
		v.NextNode = &nv
	}
	rows, err := storage.ListProgress(ctx, db, s.LearnerID)
	if err != nil {
		return v, err
	}
	for _, r := range rows {
		n, ok := s.Cur.Node(r.NodeID)
		if !ok || r.Status != string(progress.StatusMastered) || r.MasteredAt == nil {
			continue
		}
		v.Mastered = append(v.Mastered, MasteredItem{n.ID, n.Title, n.Level, r.MasteryScore, r.MasteredAt.Format("2006-01-02T15:04:05Z07:00")})
	}
	attempts, err := storage.RecentAttempts(ctx, db, s.LearnerID, 10)
	if err != nil {
		return v, err
	}
	for _, a := range attempts {
		title := a.NodeID
		if n, ok := s.Cur.Node(a.NodeID); ok {
			title = n.Title
		}
		v.Attempts = append(v.Attempts, AttemptItem{a, title})
	}
	open, err := storage.OpenSessions(ctx, db, s.LearnerID)
	if err != nil {
		return v, err
	}
	for _, o := range open {
		if n, ok := s.Cur.Node(o.NodeID); ok {
			v.OpenSessions = append(v.OpenSessions, OpenSessionItem{o.ID, n.ID, n.Title, o.Phase})
		}
	}
	v.Weekly.Messages, v.Weekly.Attempts, v.Weekly.Mastered, err = storage.WeeklyActivity(ctx, db, s.LearnerID)
	return v, err
}
