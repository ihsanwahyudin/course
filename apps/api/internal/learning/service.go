// Package learning orchestrates the learning loop for every course: catalog
// and roadmap views, lesson sessions, tutor turns, assessments, mastery and
// the dashboard. It is the only place that changes progress, and it does so
// from validated scores and fixed rules. Every call is scoped to a user.
package learning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/apperr"
	"fluentpath/api/internal/assessment"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/progress"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

const MaxMessageRunes = 2000

// Error is the shared client-facing error type.
type Error = apperr.Error

func errf(status int, code string, retryable bool, format string, args ...any) *Error {
	return apperr.New(status, code, retryable, format, args...)
}

var (
	errAIUnavailable = errf(502, "tutor_unavailable", true, "The tutor is not responding right now. Please try again.")
	errBusy          = errf(409, "session_busy", true, "This lesson was updated in another tab. Please retry.")
)

type Service struct {
	Store   *storage.Store
	Catalog *curriculum.Catalog
	AI      ai.Provider
	// TimeZone decides where a "day" starts for streaks (IANA name).
	TimeZone string
}

func (s *Service) course(slug string) (*curriculum.Curriculum, error) {
	cur, ok := s.Catalog.Course(slug)
	if !ok {
		return nil, errf(404, "course_not_found", false, "No course %q.", slug)
	}
	return cur, nil
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
	Percent        int            `json:"percent"` // TargetMastered / TargetNodes
	Levels         []LevelSummary `json:"levels"`
}

type CourseProgress struct {
	Percent        int       `json:"percent"`
	Mastered       int       `json:"mastered"`
	TargetNodes    int       `json:"targetNodes"`
	CurrentLevel   string    `json:"currentLevel"`
	TargetLevel    string    `json:"targetLevel"`
	NextNode       *NodeView `json:"nextNode"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// CourseCard is a course as shown in the catalog and on the dashboard.
type CourseCard struct {
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Tagline     string          `json:"tagline"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Tone        string          `json:"tone"`
	Status      string          `json:"status"`
	TargetLevel string          `json:"targetLevel"`
	Levels      []string        `json:"levels"`
	TotalNodes  int             `json:"totalNodes"`
	ReadyNodes  int             `json:"readyNodes"`
	Enrolled    bool            `json:"enrolled"`
	Progress    *CourseProgress `json:"progress"`
}

type EnrollmentView struct {
	CurrentLevel string `json:"currentLevel"`
	TargetLevel  string `json:"targetLevel"`
}

type RoadmapView struct {
	Course        CourseCard     `json:"course"`
	Authenticated bool           `json:"authenticated"`
	Enrolled      bool           `json:"enrolled"`
	Enrollment    EnrollmentView `json:"enrollment"`
	Version       string         `json:"version"`
	Nodes         []NodeView     `json:"nodes"`
	NextNodeID    *string        `json:"nextNodeId"`
	Summary       Summary        `json:"summary"`
}

// state is one user's view of one course. userID "" is an anonymous
// visitor, who sees the statuses of a brand-new learner.
type state struct {
	cur        *curriculum.Curriculum
	userID     string
	enrolled   bool
	enrollment storage.Enrollment
	stored     map[string]progress.Stored
	statuses   map[string]progress.Status
}

func (s *Service) load(ctx context.Context, db storage.DB, userID string, cur *curriculum.Curriculum) (state, error) {
	st := state{
		cur: cur, userID: userID, stored: map[string]progress.Stored{},
		enrollment: storage.Enrollment{CourseSlug: cur.Slug, CurrentLevel: cur.LevelIDs[0], TargetLevel: cur.TargetLevel},
	}
	if userID != "" {
		e, err := storage.GetEnrollment(ctx, db, userID, cur.Slug)
		switch {
		case err == nil:
			st.enrolled, st.enrollment = true, e
		case !errors.Is(err, storage.ErrNotFound):
			return st, err
		}
		rows, err := storage.ListProgress(ctx, db, userID, cur.Slug)
		if err != nil {
			return st, err
		}
		for _, r := range rows {
			st.stored[r.NodeID] = progress.Stored{Status: r.Status, MasteryScore: r.MasteryScore}
		}
	}
	st.statuses = progress.Derive(cur, st.stored)
	return st, nil
}

func nodeView(st state, n *curriculum.Node) NodeView {
	succ := st.cur.Successors(n.ID)
	if succ == nil {
		succ = []string{}
	}
	return NodeView{Node: n, Status: st.statuses[n.ID], MasteryScore: st.stored[n.ID].MasteryScore, SuccessorIDs: succ}
}

func summary(st state) Summary {
	sum := Summary{}
	target := st.cur.LevelIndex(st.enrollment.TargetLevel)
	byLevel := map[string]*LevelSummary{}
	for _, li := range st.cur.Levels {
		byLevel[li.Level] = &LevelSummary{Level: li.Level, Title: li.Title, Description: li.Description}
	}
	for _, n := range st.cur.Nodes {
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
		if st.cur.LevelIndex(n.Level) <= target {
			sum.TargetNodes++
			if mastered {
				sum.TargetMastered++
			}
		}
	}
	for _, li := range st.cur.Levels {
		sum.Levels = append(sum.Levels, *byLevel[li.Level])
	}
	if sum.TargetNodes > 0 {
		sum.Percent = int(math.Round(float64(sum.TargetMastered) * 100 / float64(sum.TargetNodes)))
	}
	return sum
}

func courseCard(st state) CourseCard {
	c := st.cur
	sum := summary(st)
	card := CourseCard{
		Slug: c.Slug, Title: c.Title, Tagline: c.Tagline, Description: c.Description, Category: c.Category,
		Tone: c.Tone, Status: c.Status, TargetLevel: c.TargetLevel, Levels: c.LevelIDs,
		TotalNodes: sum.TotalNodes, ReadyNodes: sum.ReadyNodes, Enrolled: st.enrolled,
	}
	if st.enrolled {
		p := &CourseProgress{
			Percent: sum.Percent, Mastered: sum.MasteredNodes, TargetNodes: sum.TargetNodes,
			CurrentLevel: st.enrollment.CurrentLevel, TargetLevel: st.enrollment.TargetLevel,
			LastActivityAt: st.enrollment.LastActivityAt,
		}
		if next := progress.NextSuggested(c, st.statuses); next != nil {
			nv := nodeView(st, next)
			p.NextNode = &nv
		}
		card.Progress = p
	}
	return card
}

// Courses lists every course; for a logged-in user, enrolled courses carry progress.
func (s *Service) Courses(ctx context.Context, userID string) ([]CourseCard, error) {
	out := make([]CourseCard, 0, len(s.Catalog.Courses))
	for _, cur := range s.Catalog.Courses {
		st, err := s.load(ctx, s.Store.Pool, userID, cur)
		if err != nil {
			return nil, err
		}
		out = append(out, courseCard(st))
	}
	return out, nil
}

func (s *Service) Roadmap(ctx context.Context, userID, courseSlug string) (RoadmapView, error) {
	cur, err := s.course(courseSlug)
	if err != nil {
		return RoadmapView{}, err
	}
	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return RoadmapView{}, err
	}
	v := RoadmapView{
		Course: courseCard(st), Authenticated: userID != "", Enrolled: st.enrolled,
		Enrollment: EnrollmentView{st.enrollment.CurrentLevel, st.enrollment.TargetLevel},
		Version:    cur.Version, Summary: summary(st),
	}
	for _, n := range cur.Nodes {
		v.Nodes = append(v.Nodes, nodeView(st, n))
	}
	if next := progress.NextSuggested(cur, st.statuses); next != nil {
		v.NextNodeID = &next.ID
	}
	return v, nil
}

// Enroll adds the course to the user's dashboard (idempotent).
func (s *Service) Enroll(ctx context.Context, userID, courseSlug string) (RoadmapView, error) {
	cur, err := s.course(courseSlug)
	if err != nil {
		return RoadmapView{}, err
	}
	if cur.Status != curriculum.CoursePublished {
		return RoadmapView{}, errf(409, "course_coming_soon", false, "%s isn't open yet.", cur.Title)
	}
	if _, err := storage.EnsureEnrollment(ctx, s.Store.Pool, userID, cur.Slug, cur.LevelIDs[0], cur.TargetLevel); err != nil {
		return RoadmapView{}, err
	}
	return s.Roadmap(ctx, userID, courseSlug)
}

type CourseRef struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type NodeDetail struct {
	NodeView
	Course        CourseRef  `json:"course"`
	Prerequisites []NodeView `json:"prerequisites"`
	Successors    []NodeView `json:"successors"`
	CanStart      bool       `json:"canStart"`
	BlockedReason string     `json:"blockedReason,omitempty"`
	OpenSessionID *string    `json:"openSessionId"`
	KeyPoints     []string   `json:"keyPoints"`
}

func (s *Service) Node(ctx context.Context, userID, courseSlug, id string) (NodeDetail, error) {
	cur, err := s.course(courseSlug)
	if err != nil {
		return NodeDetail{}, err
	}
	n, ok := cur.Node(id)
	if !ok {
		return NodeDetail{}, errf(404, "node_not_found", false, "No roadmap node %q.", id)
	}
	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return NodeDetail{}, err
	}
	d := NodeDetail{NodeView: nodeView(st, n), Course: CourseRef{cur.Slug, cur.Title}, Prerequisites: []NodeView{}, Successors: []NodeView{}, KeyPoints: []string{}}
	for _, p := range n.PrerequisiteIDs {
		pn, _ := cur.Node(p)
		d.Prerequisites = append(d.Prerequisites, nodeView(st, pn))
	}
	for _, c := range cur.Successors(n.ID) {
		cn, _ := cur.Node(c)
		d.Successors = append(d.Successors, nodeView(st, cn))
	}
	d.CanStart, d.BlockedReason = startable(st, n)
	if l, ok := cur.Lesson(n.ID); ok {
		for _, k := range l.KeyPoints {
			d.KeyPoints = append(d.KeyPoints, k.Title)
		}
	}
	if userID != "" {
		if sess, err := storage.FindOpenSession(ctx, s.Store.Pool, userID, cur.Slug, n.ID); err == nil {
			d.OpenSessionID = &sess.ID
		} else if !errors.Is(err, storage.ErrNotFound) {
			return NodeDetail{}, err
		}
	}
	return d, nil
}

func startable(st state, n *curriculum.Node) (bool, string) {
	switch st.statuses[n.ID] {
	case progress.StatusComingSoon:
		return false, "This lesson is coming soon."
	case progress.StatusLocked:
		var missing []string
		for _, p := range n.PrerequisiteIDs {
			if st.statuses[p] != progress.StatusMastered {
				pn, _ := st.cur.Node(p)
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
	Course      CourseRef            `json:"course"`
	Node        NodeView             `json:"node"`
	Messages    []storage.Message    `json:"messages"`
	Steps       []tutor.Phase        `json:"steps"`
	CanChat     bool                 `json:"canChat"`
	Assessment  []AssessmentItemView `json:"assessment"`
	LastAttempt *storage.Attempt     `json:"lastAttempt"`
}

func (s *Service) sessionCourse(sess storage.Session) (*curriculum.Curriculum, *curriculum.Node, error) {
	cur, ok := s.Catalog.Course(sess.CourseSlug)
	if !ok {
		return nil, nil, errf(410, "course_removed", false, "This course is no longer available.")
	}
	n, ok := cur.Node(sess.NodeID)
	if !ok {
		return nil, nil, errf(410, "node_removed", false, "This lesson is no longer in the course.")
	}
	return cur, n, nil
}

func (s *Service) sessionView(ctx context.Context, db storage.DB, sess storage.Session) (SessionView, error) {
	cur, n, err := s.sessionCourse(sess)
	if err != nil {
		return SessionView{}, err
	}
	st, err := s.load(ctx, db, sess.UserID, cur)
	if err != nil {
		return SessionView{}, err
	}
	msgs, err := storage.ListMessages(ctx, db, sess.ID)
	if err != nil {
		return SessionView{}, err
	}
	v := SessionView{
		Session: sess, Course: CourseRef{cur.Slug, cur.Title}, Node: nodeView(st, n), Messages: msgs, Steps: tutor.Steps,
		CanChat:    sess.EndedAt == nil && tutor.Phase(sess.Phase).AcceptsChat(),
		Assessment: []AssessmentItemView{},
	}
	if sess.Phase == string(tutor.PhaseAssessment) {
		if l, ok := cur.Lesson(n.ID); ok {
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

// StartSession opens (or resumes) the user's session for a ready node,
// enrolling them in the course if needed.
func (s *Service) StartSession(ctx context.Context, userID, courseSlug, nodeID string) (SessionView, error) {
	cur, err := s.course(courseSlug)
	if err != nil {
		return SessionView{}, err
	}
	n, ok := cur.Node(nodeID)
	if !ok {
		return SessionView{}, errf(404, "node_not_found", false, "No roadmap node %q.", nodeID)
	}
	if cur.Status != curriculum.CoursePublished {
		return SessionView{}, errf(409, "course_coming_soon", false, "%s isn't open yet.", cur.Title)
	}
	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return SessionView{}, err
	}
	if ok, reason := startable(st, n); !ok {
		code := "node_locked"
		if st.statuses[n.ID] == progress.StatusComingSoon {
			code = "node_coming_soon"
		}
		return SessionView{}, errf(409, code, false, "%s", reason)
	}
	lesson, _ := cur.Lesson(n.ID)

	sess, err := storage.FindOpenSession(ctx, s.Store.Pool, userID, cur.Slug, n.ID)
	if errors.Is(err, storage.ErrNotFound) {
		err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
			if _, err := storage.EnsureEnrollment(ctx, tx, userID, cur.Slug, cur.LevelIDs[0], cur.TargetLevel); err != nil {
				return err
			}
			var err error
			sess, err = storage.CreateSession(ctx, tx, userID, cur.Slug, n.ID, string(tutor.PhaseIntroduction))
			if err != nil {
				return err
			}
			if _, err := storage.AddMessage(ctx, tx, sess.ID, "assistant", lesson.Intro, sess.Phase, map[string]any{"kind": "intro"}); err != nil {
				return err
			}
			if err := storage.MarkInProgress(ctx, tx, userID, cur.Slug, n.ID); err != nil {
				return err
			}
			return storage.TouchEnrollment(ctx, tx, userID, cur.Slug)
		})
		if errors.Is(err, storage.ErrConflict) {
			sess, err = storage.FindOpenSession(ctx, s.Store.Pool, userID, cur.Slug, n.ID)
		}
	}
	if err != nil {
		return SessionView{}, err
	}
	return s.sessionView(ctx, s.Store.Pool, sess)
}

func (s *Service) getOwnSession(ctx context.Context, userID, id string) (storage.Session, error) {
	sess, err := storage.GetSession(ctx, s.Store.Pool, id, userID)
	if errors.Is(err, storage.ErrNotFound) {
		// Same answer for "doesn't exist" and "belongs to someone else".
		return sess, errf(404, "session_not_found", false, "Session not found.")
	}
	return sess, err
}

func (s *Service) GetSession(ctx context.Context, userID, id string) (SessionView, error) {
	sess, err := s.getOwnSession(ctx, userID, id)
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
func (s *Service) SendMessage(ctx context.Context, userID, sessionID, content string) (SessionView, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return SessionView{}, errf(400, "empty_message", false, "Message cannot be empty.")
	}
	if utf8.RuneCountInString(content) > MaxMessageRunes {
		return SessionView{}, errf(400, "message_too_long", false, "Message is too long (max %d characters).", MaxMessageRunes)
	}
	sess, err := s.getOwnSession(ctx, userID, sessionID)
	if err != nil {
		return SessionView{}, err
	}
	phase := tutor.Phase(sess.Phase)
	if sess.EndedAt != nil || !phase.AcceptsChat() {
		return SessionView{}, errf(409, "wrong_phase", false, "This lesson is in %s and doesn't accept chat messages.", phase)
	}
	cur, n, err := s.sessionCourse(sess)
	if err != nil {
		return SessionView{}, err
	}
	lesson, _ := cur.Lesson(n.ID)

	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return SessionView{}, err
	}
	history, err := storage.ListMessages(ctx, s.Store.Pool, sess.ID)
	if err != nil {
		return SessionView{}, err
	}
	in := ai.TutorInput{
		Policy:       cur.Tutor.Policy,
		CurrentLevel: st.enrollment.CurrentLevel, TargetLevel: st.enrollment.TargetLevel,
		Node: n, Lesson: lesson, Phase: phase, NextPhase: tutor.NextOf(phase),
		TurnsInPhase: sess.PhaseTurns + 1,
	}
	for _, m := range history {
		if m.Role == "user" || m.Role == "assistant" {
			in.Messages = append(in.Messages, ai.ChatMessage{Role: m.Role, Content: m.Content})
		}
	}
	in.Messages = append(in.Messages, ai.ChatMessage{Role: "user", Content: content})
	for _, cn := range cur.Nodes {
		if st.statuses[cn.ID] == progress.StatusMastered {
			in.MasteredNodeTitles = append(in.MasteredNodeTitles, cn.Title)
		}
	}
	in.ObservedWeaknesses, err = s.weaknesses(ctx, sess, n)
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
		return storage.TouchEnrollment(ctx, tx, userID, cur.Slug)
	})
	if errors.Is(err, storage.ErrConflict) {
		return SessionView{}, errBusy
	}
	if err != nil {
		return SessionView{}, err
	}
	return s.GetSession(ctx, userID, sess.ID)
}

// weaknesses combines recent tutor-observed mistake tags with objectives that
// scored low on this session's last attempt.
func (s *Service) weaknesses(ctx context.Context, sess storage.Session, n *curriculum.Node) ([]string, error) {
	tags, err := storage.RecentMistakes(ctx, s.Store.Pool, sess.UserID, sess.CourseSlug, 5)
	if err != nil {
		return nil, err
	}
	if a, err := storage.LatestAttempt(ctx, s.Store.Pool, sess.ID); err == nil && !a.Passed {
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

func (s *Service) SubmitAssessment(ctx context.Context, userID, sessionID string, answers map[string]string) (AssessmentResult, error) {
	sess, err := s.getOwnSession(ctx, userID, sessionID)
	if err != nil {
		return AssessmentResult{}, err
	}
	if sess.Phase == string(tutor.PhaseMastered) {
		return s.completedResult(ctx, userID, sess)
	}
	if sess.Phase != string(tutor.PhaseAssessment) {
		return AssessmentResult{}, errf(409, "wrong_phase", false, "This lesson isn't ready for assessment yet (current phase: %s).", sess.Phase)
	}
	cur, n, err := s.sessionCourse(sess)
	if err != nil {
		return AssessmentResult{}, err
	}
	lesson, _ := cur.Lesson(n.ID)
	clean := map[string]string{}
	for k, v := range answers {
		clean[k] = strings.TrimSpace(v)
	}
	if err := assessment.ValidateAnswers(lesson.Assessment, clean); err != nil {
		return AssessmentResult{}, errf(400, "invalid_answers", false, "%s", err.Error())
	}

	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return AssessmentResult{}, err
	}
	raw, err := s.AI.Evaluate(ctx, ai.EvalInput{Policy: cur.Tutor.EvaluatorPolicy, TargetLevel: st.enrollment.TargetLevel, Node: n, Items: lesson.Assessment, Answers: clean})
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
		locked, err := storage.LockSession(ctx, tx, sess.ID, userID)
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
		before, err := s.load(ctx, tx, userID, cur)
		if err != nil {
			return err
		}
		if _, err := storage.LockProgress(ctx, tx, userID, cur.Slug, n.ID); err != nil {
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
		out.Attempt.CourseSlug, out.Attempt.NodeID = cur.Slug, n.ID
		out.Passed = res.Passed
		if res.Passed {
			if out.NewlyMastered, err = storage.MarkMastered(ctx, tx, userID, cur.Slug, n.ID, res.Score); err != nil {
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
	out.Session, err = s.GetSession(ctx, userID, sess.ID)
	return out, err
}

// completedResult answers a repeated submission for an already-mastered
// session without re-scoring or re-awarding anything.
func (s *Service) completedResult(ctx context.Context, userID string, sess storage.Session) (AssessmentResult, error) {
	view, err := s.GetSession(ctx, userID, sess.ID)
	if err != nil {
		return AssessmentResult{}, err
	}
	out := AssessmentResult{AlreadyCompleted: true, Passed: true, Unlocked: []NodeView{}, Session: view}
	if view.LastAttempt != nil {
		out.Attempt = *view.LastAttempt
	}
	return out, nil
}

// ---- progress and dashboard ----

type MasteredItem struct {
	NodeID     string    `json:"nodeId"`
	Title      string    `json:"title"`
	Level      string    `json:"level"`
	Score      *int      `json:"score"`
	MasteredAt time.Time `json:"masteredAt"`
}

type AttemptItem struct {
	storage.Attempt
	Title       string                 `json:"title"`
	CourseTitle string                 `json:"courseTitle"`
	Objectives  []curriculum.Objective `json:"objectives"`
}

type OpenSessionItem struct {
	SessionID   string    `json:"sessionId"`
	CourseSlug  string    `json:"courseSlug"`
	CourseTitle string    `json:"courseTitle"`
	NodeID      string    `json:"nodeId"`
	Title       string    `json:"title"`
	Phase       string    `json:"phase"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Weekly struct {
	Messages int `json:"messages"`
	Attempts int `json:"attempts"`
	Mastered int `json:"mastered"`
}

type ProgressView struct {
	Course       CourseCard        `json:"course"`
	Summary      Summary           `json:"summary"`
	Mastered     []MasteredItem    `json:"mastered"`
	Attempts     []AttemptItem     `json:"recentAttempts"`
	OpenSessions []OpenSessionItem `json:"openSessions"`
}

func (s *Service) attemptItems(attempts []storage.Attempt) []AttemptItem {
	out := make([]AttemptItem, 0, len(attempts))
	for _, a := range attempts {
		item := AttemptItem{Attempt: a, Title: a.NodeID, CourseTitle: a.CourseSlug, Objectives: []curriculum.Objective{}}
		if cur, ok := s.Catalog.Course(a.CourseSlug); ok {
			item.CourseTitle = cur.Title
			if n, ok := cur.Node(a.NodeID); ok {
				item.Title, item.Objectives = n.Title, n.Objectives
			}
		}
		out = append(out, item)
	}
	return out
}

func (s *Service) openSessionItems(sessions []storage.Session, onlyCourse string) []OpenSessionItem {
	out := []OpenSessionItem{}
	for _, o := range sessions {
		if onlyCourse != "" && o.CourseSlug != onlyCourse {
			continue
		}
		cur, ok := s.Catalog.Course(o.CourseSlug)
		if !ok {
			continue
		}
		if n, ok := cur.Node(o.NodeID); ok {
			out = append(out, OpenSessionItem{o.ID, cur.Slug, cur.Title, n.ID, n.Title, o.Phase, o.UpdatedAt})
		}
	}
	return out
}

func (s *Service) CourseProgress(ctx context.Context, userID, courseSlug string) (ProgressView, error) {
	cur, err := s.course(courseSlug)
	if err != nil {
		return ProgressView{}, err
	}
	db := s.Store.Pool
	st, err := s.load(ctx, db, userID, cur)
	if err != nil {
		return ProgressView{}, err
	}
	v := ProgressView{Course: courseCard(st), Summary: summary(st), Mastered: []MasteredItem{}}
	rows, err := storage.ListProgress(ctx, db, userID, cur.Slug)
	if err != nil {
		return v, err
	}
	for _, r := range rows {
		n, ok := cur.Node(r.NodeID)
		if ok && r.Status == string(progress.StatusMastered) && r.MasteredAt != nil {
			v.Mastered = append(v.Mastered, MasteredItem{n.ID, n.Title, n.Level, r.MasteryScore, *r.MasteredAt})
		}
	}
	sort.Slice(v.Mastered, func(i, j int) bool { return v.Mastered[i].MasteredAt.After(v.Mastered[j].MasteredAt) })
	attempts, err := storage.RecentAttempts(ctx, db, userID, cur.Slug, 10)
	if err != nil {
		return v, err
	}
	v.Attempts = s.attemptItems(attempts)
	open, err := storage.OpenSessions(ctx, db, userID)
	if err != nil {
		return v, err
	}
	v.OpenSessions = s.openSessionItems(open, cur.Slug)
	return v, nil
}

type DayActivity struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Streak struct {
	Current     int  `json:"current"`
	ActiveToday bool `json:"activeToday"`
}

type DashboardView struct {
	User           storage.User      `json:"user"`
	Courses        []CourseCard      `json:"courses"` // enrolled, most recent activity first
	Explore        []CourseCard      `json:"explore"` // not enrolled yet
	OpenSessions   []OpenSessionItem `json:"openSessions"`
	RecentAttempts []AttemptItem     `json:"recentAttempts"`
	Weekly         Weekly            `json:"weekly"`
	Streak         Streak            `json:"streak"`
	Activity       []DayActivity     `json:"activity"` // last 28 days, oldest first
	TotalMastered  int               `json:"totalMastered"`
}

const activityDays = 28

func (s *Service) Dashboard(ctx context.Context, user storage.User) (DashboardView, error) {
	db := s.Store.Pool
	v := DashboardView{User: user, Courses: []CourseCard{}, Explore: []CourseCard{}}
	enrollments, err := storage.ListEnrollments(ctx, db, user.ID)
	if err != nil {
		return v, err
	}
	enrolled := map[string]bool{}
	for _, e := range enrollments {
		cur, ok := s.Catalog.Course(e.CourseSlug)
		if !ok {
			continue
		}
		enrolled[cur.Slug] = true
		st, err := s.load(ctx, db, user.ID, cur)
		if err != nil {
			return v, err
		}
		card := courseCard(st)
		v.TotalMastered += card.Progress.Mastered
		v.Courses = append(v.Courses, card)
	}
	for _, cur := range s.Catalog.Courses {
		if !enrolled[cur.Slug] {
			st, err := s.load(ctx, db, "", cur)
			if err != nil {
				return v, err
			}
			v.Explore = append(v.Explore, courseCard(st))
		}
	}
	open, err := storage.OpenSessions(ctx, db, user.ID)
	if err != nil {
		return v, err
	}
	v.OpenSessions = s.openSessionItems(open, "")
	attempts, err := storage.RecentAttempts(ctx, db, user.ID, "", 6)
	if err != nil {
		return v, err
	}
	v.RecentAttempts = s.attemptItems(attempts)
	if v.Weekly.Messages, v.Weekly.Attempts, v.Weekly.Mastered, err = storage.WeeklyActivity(ctx, db, user.ID); err != nil {
		return v, err
	}

	loc, err := time.LoadLocation(s.TimeZone)
	if err != nil || s.TimeZone == "" {
		loc = time.UTC
	}
	days, err := storage.DailyActivity(ctx, db, user.ID, loc.String(), 60)
	if err != nil {
		return v, err
	}
	v.Activity, v.Streak = activity(days, time.Now().In(loc))
	return v, nil
}

// activity turns per-day counts into the last-28-days grid and the current
// streak. A streak stays alive until a full day passes without activity.
func activity(days map[string]int, now time.Time) ([]DayActivity, Streak) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	key := func(t time.Time) string { return t.Format("2006-01-02") }
	grid := make([]DayActivity, 0, activityDays)
	for i := activityDays - 1; i >= 0; i-- {
		d := today.AddDate(0, 0, -i)
		grid = append(grid, DayActivity{key(d), days[key(d)]})
	}
	st := Streak{ActiveToday: days[key(today)] > 0}
	d := today
	if !st.ActiveToday {
		d = today.AddDate(0, 0, -1)
	}
	for days[key(d)] > 0 {
		st.Current++
		d = d.AddDate(0, 0, -1)
	}
	return grid, st
}
