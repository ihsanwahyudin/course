package learning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

const (
	maxAsksPerStep = 20
	planCacheTTL   = cacheTTL
)

// ---- per-step state (stored as JSON on the session) ----

type qState struct {
	Hints    []string `json:"hints"`
	Choice   *int     `json:"choice,omitempty"`
	Tokens   []string `json:"tokens,omitempty"`
	Answered bool     `json:"answered"`
	Correct  bool     `json:"correct"`
	Revealed bool     `json:"revealed"`
}

type askEntry struct {
	Question    string `json:"question"`
	Answer      string `json:"answer"`
	Scope       string `json:"scope"`
	OtherLesson string `json:"otherLesson,omitempty"`
}

type similarState struct {
	Question tutor.Question `json:"question"`
	qState
}

type stepState struct {
	qState
	Asks    []askEntry    `json:"asks"`
	Similar *similarState `json:"similar,omitempty"`
}

// ---- views (answers only after the learner has answered or given up) ----

type QuestionView struct {
	Type        string   `json:"type"`
	ObjectiveID string   `json:"objectiveId"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options"`
	Tokens      []string `json:"tokens"`
	Hints       []string `json:"hints"`
	HintsLeft   int      `json:"hintsLeft"`
	Choice      *int     `json:"choice"`
	Picked      []string `json:"picked"`
	Answered    bool     `json:"answered"`
	Correct     bool     `json:"correct"`
	Revealed    bool     `json:"revealed"`
	Answer      *int     `json:"answer"`
	Solution    string   `json:"solution"`
	Explanation string   `json:"explanation"`
}

type CuratedLink struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Source  string `json:"source"`
	Note    string `json:"note,omitempty"`
	VideoID string `json:"videoId,omitempty"`
}

type StepView struct {
	Index      int             `json:"index"`
	Kind       tutor.StepKind  `json:"kind"`
	Title      string          `json:"title"`
	Text       string          `json:"text"`
	Body       string          `json:"body"`
	Pairs      []tutor.Pair    `json:"pairs"`
	Examples   []tutor.Example `json:"examples"`
	Timeline   *tutor.Timeline `json:"timeline"`
	Flashcards []tutor.Card    `json:"flashcards"`
	Artifact   *tutor.Artifact `json:"artifact"`
	Resources  []CuratedLink   `json:"resources"`
	Question   *QuestionView   `json:"question"`
	Similar    *QuestionView   `json:"similar"`
	Asks       []askEntry      `json:"asks"`
	Done       bool            `json:"done"`
}

type MemoryChip struct {
	From   string `json:"from"` // previous lesson title
	Strong string `json:"strong"`
	Weak   string `json:"weak"`
}

type SummaryView struct {
	Mastered        []string  `json:"mastered"`
	Review          []string  `json:"review"`
	Remembered      []string  `json:"remembered"`
	FlashcardsAdded int       `json:"flashcardsAdded"`
	NextOpening     string    `json:"nextOpening"`
	Next            *NodeView `json:"next"`
}

// LessonView is the step-by-step lesson as the learner sees it.
type LessonView struct {
	Greeting   string       `json:"greeting"`
	Steps      []StepView   `json:"steps"` // revealed so far
	StepIndex  int          `json:"stepIndex"`
	TotalSteps int          `json:"totalSteps"`
	Checkpoint bool         `json:"checkpoint"` // every step done: the checkpoint is next
	MemoryChip *MemoryChip  `json:"memoryChip"`
	Summary    *SummaryView `json:"summary"`
}

func questionView(q tutor.Question, st qState) *QuestionView {
	v := &QuestionView{Type: q.Type, ObjectiveID: q.ObjectiveID, Prompt: q.Prompt, Options: q.Options, Tokens: q.Tokens,
		Hints: st.Hints, HintsLeft: tutor.MaxHints - len(st.Hints), Choice: st.Choice, Picked: st.Tokens,
		Answered: st.Answered, Correct: st.Correct, Revealed: st.Revealed}
	if v.Hints == nil {
		v.Hints = []string{}
	}
	if v.Options == nil {
		v.Options = []string{}
	}
	if v.Tokens == nil {
		v.Tokens = []string{}
	}
	if st.Answered || st.Revealed {
		if q.Type == tutor.QuestionMCQ {
			a := q.Answer
			v.Answer = &a
		}
		v.Solution, v.Explanation = q.Solution, q.Explanation
	}
	return v
}

// ---- plan loading ----

type lessonCtx struct {
	cur    *curriculum.Curriculum
	node   *curriculum.Node
	lesson *curriculum.Lesson
	sess   storage.Session
	plan   tutor.Plan
	states []stepState
}

func (lc *lessonCtx) state(i int) *stepState {
	for len(lc.states) <= i {
		lc.states = append(lc.states, stepState{})
	}
	return &lc.states[i]
}

func decodeLesson(cur *curriculum.Curriculum, n *curriculum.Node, sess storage.Session) (*lessonCtx, error) {
	l, _ := cur.Lesson(n.ID)
	lc := &lessonCtx{cur: cur, node: n, lesson: l, sess: sess}
	if len(sess.Plan) == 0 {
		return lc, nil
	}
	if err := json.Unmarshal(sess.Plan, &lc.plan); err != nil {
		return nil, fmt.Errorf("decode plan: %w", err)
	}
	if len(sess.StepState) > 0 {
		if err := json.Unmarshal(sess.StepState, &lc.states); err != nil {
			return nil, fmt.Errorf("decode step state: %w", err)
		}
	}
	return lc, nil
}

func (s *Service) lessonInput(ctx context.Context, st state, n *curriculum.Node, lang string) ai.LessonInput {
	cur := st.cur
	l, _ := cur.Lesson(n.ID)
	in := ai.LessonInput{
		Policy: cur.Tutor.Policy, Locale: lang, LanguagePolicy: cur.Tutor.LanguagePolicy[lang],
		CurrentLevel: st.enrollment.CurrentLevel, TargetLevel: st.enrollment.TargetLevel,
		Node: cur.LocalNode(n.ID, lang), Lesson: l, Material: cur.MaterialIn(n.ID, lang),
		CacheKey: fmt.Sprintf("%s:%s:%s", cur.Slug, n.ID, lang),
	}
	in.LearnerProfile, in.LearnerNotes = cur.TutorHints(st.profile)
	if st.userID != "" {
		if items, err := storage.ListMemory(ctx, s.Store.Pool, st.userID, 10); err == nil {
			for _, m := range items {
				in.Memory = append(in.Memory, m.Text)
			}
		}
	}
	return in
}

func objectiveIDs(n *curriculum.Node) []string {
	ids := make([]string, len(n.Objectives))
	for i, o := range n.Objectives {
		ids[i] = o.ID
	}
	return ids
}

// ensurePlan writes the lesson plan the first time a session needs one: the
// AI's plan if it validates (one retry), otherwise one built from the
// material. Plans are cached per lesson, language, goal and memory.
func (s *Service) ensurePlan(ctx context.Context, st state, n *curriculum.Node, sess *storage.Session) error {
	if len(sess.Plan) > 0 {
		return nil
	}
	lang := st.cur.Lang(sess.Locale)
	in := s.lessonInput(ctx, st, n, lang)
	h := sha256.Sum256([]byte(strings.Join(in.Memory, "\x00") + "\x00" + strings.Join(in.LearnerNotes, "\x00")))
	key := fmt.Sprintf("plan:v1:%s:%s:%s:%s:%s:%s", st.cur.Slug, st.cur.Version, n.ID, lang, st.profile[curriculum.GoalQuestion], hex.EncodeToString(h[:6]))
	objs := objectiveIDs(n)
	var plan tutor.Plan
	got := false
	if raw, ok, err := storage.GetCache(ctx, s.Store.Pool, key); err == nil && ok {
		if p, err := tutor.ParsePlan(raw, objs); err == nil {
			plan, got = p, true
		}
	}
	for attempt := 0; !got && attempt < 2; attempt++ {
		if err := s.checkQuota(ctx, st.userID); err != nil {
			break
		}
		res, err := s.AI.LessonPlan(ctx, in)
		s.recordUsage(ctx, st.userID, "plan", res.Usage)
		if err != nil {
			slog.Warn("lesson plan call failed", "provider", s.AI.Name(), "node", n.ID, "err", err)
			break
		}
		if p, err := tutor.ParsePlan(res.Raw, objs); err == nil {
			plan, got = p, true
			if raw, err := json.Marshal(plan); err == nil {
				_ = storage.PutCache(ctx, s.Store.Pool, key, raw, planCacheTTL)
			}
		} else {
			slog.Warn("lesson plan rejected", "provider", s.AI.Name(), "node", n.ID, "attempt", attempt, "err", err)
		}
	}
	if !got {
		plan = tutor.FallbackPlan(in.Node, in.Lesson, in.Material, lang)
		if err := tutor.ValidatePlan(&plan, objs); err != nil {
			return fmt.Errorf("fallback plan for %s: %w", n.ID, err)
		}
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	if err := storage.SetSessionPlan(ctx, s.Store.Pool, sess.ID, raw); err != nil {
		return err
	}
	sess.Plan, sess.StepIndex, sess.StepState = raw, 0, json.RawMessage(`[]`)
	return nil
}

// lessonView builds what the learner sees: steps up to the current one.
func (s *Service) lessonView(ctx context.Context, db storage.DB, st state, lc *lessonCtx, lang string) (*LessonView, error) {
	if len(lc.plan.Steps) == 0 {
		return nil, nil
	}
	v := &LessonView{Greeting: lc.plan.Greeting, StepIndex: lc.sess.StepIndex, TotalSteps: len(lc.plan.Steps), Steps: []StepView{}}
	v.Checkpoint = lc.sess.StepIndex >= len(lc.plan.Steps)
	mat := st.cur.MaterialIn(lc.node.ID, lang)
	for i, step := range lc.plan.Steps {
		if i > lc.sess.StepIndex {
			break
		}
		ss := lc.state(i)
		sv := StepView{Index: i, Kind: step.Kind, Title: step.Title, Text: step.Text, Body: step.Body, Pairs: step.Pairs, Examples: step.Examples,
			Timeline: step.Timeline, Flashcards: step.Flashcards, Artifact: step.Artifact, Asks: ss.Asks, Done: i < lc.sess.StepIndex}
		if sv.Asks == nil {
			sv.Asks = []askEntry{}
		}
		if step.Question != nil {
			sv.Question = questionView(*step.Question, ss.qState)
		}
		if ss.Similar != nil {
			sv.Similar = questionView(ss.Similar.Question, ss.Similar.qState)
		}
		if step.Kind == tutor.StepResources && mat != nil {
			for _, r := range mat.Resources {
				link := CuratedLink{Kind: r.Kind, Title: r.Title, URL: r.URL, Source: r.Source, Note: r.Note}
				if r.Kind == curriculum.ResourceVideo {
					link.VideoID, _ = curriculum.YouTubeID(r.URL)
				}
				sv.Resources = append(sv.Resources, link)
			}
		}
		v.Steps = append(v.Steps, sv)
	}
	// Where the previous lesson left off.
	if node, raw, err := storage.LatestSummary(ctx, db, lc.sess.UserID, lc.sess.CourseSlug, lc.node.ID); err == nil && raw != nil {
		var sum tutor.Summary
		if json.Unmarshal(raw, &sum) == nil {
			chip := &MemoryChip{From: st.cur.LocalNode(node, lang).Title}
			if len(sum.Mastered) > 0 {
				chip.Strong = sum.Mastered[0]
			}
			if len(sum.Review) > 0 {
				chip.Weak = sum.Review[0]
			}
			if chip.Strong != "" || chip.Weak != "" {
				v.MemoryChip = chip
			}
		}
	} else if err != nil {
		return nil, err
	}
	if len(lc.sess.Summary) > 0 {
		var sum storedSummary
		if json.Unmarshal(lc.sess.Summary, &sum) == nil {
			sv := &SummaryView{Mastered: sum.Mastered, Review: sum.Review, Remembered: sum.Memory, FlashcardsAdded: sum.FlashcardsAdded, NextOpening: sum.NextOpening}
			if sum.NextNodeID != "" {
				if nn, ok := st.cur.Node(sum.NextNodeID); ok {
					nv := nodeView(st, nn)
					sv.Next = &nv
				}
			}
			v.Summary = sv
		}
	}
	return v, nil
}

// ---- step actions ----

// stepAction locks the session, loads its plan and runs fn on step n.
// fn may call the AI; it runs inside the transaction so two tabs can't
// double-spend hints or answer twice.
func (s *Service) stepAction(ctx context.Context, userID, sessionID string, n int, kind string, fn func(lc *lessonCtx, st state, lang string) error) (SessionView, error) {
	sess, err := s.getOwnSession(ctx, userID, sessionID)
	if err != nil {
		return SessionView{}, err
	}
	if sess.EndedAt != nil {
		return SessionView{}, errLessonClosed
	}
	cur, node, err := s.sessionCourse(sess)
	if err != nil {
		return SessionView{}, err
	}
	err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		locked, err := storage.LockSession(ctx, tx, sess.ID, userID)
		if err != nil {
			return err
		}
		lc, err := decodeLesson(cur, node, locked)
		if err != nil {
			return err
		}
		if len(lc.plan.Steps) == 0 {
			return errf(409, "no_plan", true, "This lesson isn't ready yet. Please reload.")
		}
		if n < 0 || n > locked.StepIndex || n >= len(lc.plan.Steps) {
			return errf(409, "step_not_open", false, "That step isn't open yet.")
		}
		st, err := s.load(ctx, tx, userID, cur)
		if err != nil {
			return err
		}
		lang := cur.Lang(locked.Locale)
		if err := fn(lc, st, lang); err != nil {
			return err
		}
		raw, err := json.Marshal(lc.states)
		if err != nil {
			return err
		}
		if err := storage.SaveSteps(ctx, tx, sess.ID, lc.sess.StepIndex, raw); err != nil {
			return err
		}
		if err := storage.AddActivity(ctx, tx, userID, sess.ID, kind); err != nil {
			return err
		}
		return storage.TouchEnrollment(ctx, tx, userID, cur.Slug)
	})
	if err != nil {
		return SessionView{}, err
	}
	return s.GetSession(ctx, userID, sess.ID)
}

// Next moves past step n. A question must be answered (or given up) first.
func (s *Service) NextStep(ctx context.Context, userID, sessionID string, n int) (SessionView, error) {
	return s.stepAction(ctx, userID, sessionID, n, "next", func(lc *lessonCtx, st state, lang string) error {
		if n != lc.sess.StepIndex {
			return errf(409, "step_done", false, "You've already moved past this step.")
		}
		if q := lc.plan.Steps[n].Question; q != nil {
			if ss := lc.state(n); !ss.Answered && !ss.Revealed {
				return errf(409, "answer_first", false, "Answer the question (or give up) before moving on.")
			}
		}
		lc.sess.StepIndex = n + 1
		return nil
	})
}

type AnswerInput struct {
	Choice  *int     `json:"choice"`
	Tokens  []string `json:"tokens"`
	Similar bool     `json:"similar"` // answering the step's similar question
}

// target picks the question an action is about: the step's own or its similar one.
func target(lc *lessonCtx, n int, similar bool) (*tutor.Question, *qState, error) {
	ss := lc.state(n)
	if similar {
		if ss.Similar == nil {
			return nil, nil, errf(409, "no_similar", false, "There's no similar question here.")
		}
		return &ss.Similar.Question, &ss.Similar.qState, nil
	}
	q := lc.plan.Steps[n].Question
	if q == nil {
		return nil, nil, errf(409, "not_a_question", false, "This step has no question.")
	}
	return q, &ss.qState, nil
}

func (s *Service) AnswerStep(ctx context.Context, userID, sessionID string, n int, in AnswerInput) (SessionView, error) {
	return s.stepAction(ctx, userID, sessionID, n, "answer", func(lc *lessonCtx, st state, lang string) error {
		q, qs, err := target(lc, n, in.Similar)
		if err != nil {
			return err
		}
		if qs.Answered || qs.Revealed {
			return errf(409, "already_answered", false, "This question is already answered.")
		}
		switch q.Type {
		case tutor.QuestionMCQ:
			if in.Choice == nil || *in.Choice < 0 || *in.Choice >= len(q.Options) {
				return errf(400, "invalid_answer", false, "Pick one of the options.")
			}
			c := *in.Choice
			qs.Choice, qs.Correct = &c, c == q.Answer
		case tutor.QuestionWordOrder:
			if len(in.Tokens) == 0 || len(in.Tokens) > 20 {
				return errf(400, "invalid_answer", false, "Arrange the words first.")
			}
			for _, t := range in.Tokens {
				if utf8.RuneCountInString(t) > 40 {
					return errf(400, "invalid_answer", false, "Arrange the words first.")
				}
			}
			qs.Tokens = in.Tokens
			qs.Correct = tutor.NormalizeWords(strings.Join(in.Tokens, " ")) == tutor.NormalizeWords(q.Solution)
		}
		qs.Answered = true
		return nil
	})
}

var fallbackHints = map[string][2]string{
	"en": {"Read the sentence again and look at who or what it's about.", "Compare each option with the rule you just learnt."},
	"id": {"Baca lagi kalimatnya dan lihat siapa atau apa yang dibicarakan.", "Bandingkan setiap pilihan dengan aturan yang baru kamu pelajari."},
}

func (s *Service) HintStep(ctx context.Context, userID, sessionID string, n int, similar bool) (SessionView, error) {
	return s.stepAction(ctx, userID, sessionID, n, "hint", func(lc *lessonCtx, st state, lang string) error {
		q, qs, err := target(lc, n, similar)
		if err != nil {
			return err
		}
		if qs.Answered || qs.Revealed {
			return errf(409, "already_answered", false, "This question is already answered.")
		}
		if len(qs.Hints) >= tutor.MaxHints {
			return errf(409, "no_hints_left", false, "You've used both hints.")
		}
		hint := ""
		if s.checkQuota(ctx, userID) == nil {
			in := ai.HintInput{LessonInput: s.lessonInput(ctx, st, lc.node, lang), Question: *q, Previous: qs.Hints}
			for attempt := 0; attempt < 2 && hint == ""; attempt++ {
				res, err := s.AI.Hint(ctx, in)
				s.recordUsage(ctx, userID, "hint", res.Usage)
				if err != nil {
					slog.Warn("hint call failed", "err", err)
					break
				}
				if h, err := tutor.ParseHint(res.Raw, *q); err == nil {
					hint = h
				} else {
					slog.Warn("hint rejected", "attempt", attempt, "err", err)
				}
			}
		}
		if hint == "" {
			fh, ok := fallbackHints[lang]
			if !ok {
				fh = fallbackHints["en"]
			}
			hint = fh[len(qs.Hints)]
		}
		qs.Hints = append(qs.Hints, hint)
		return nil
	})
}

// RevealStep shows the answer after both hints were used ("I give up").
func (s *Service) RevealStep(ctx context.Context, userID, sessionID string, n int, similar bool) (SessionView, error) {
	return s.stepAction(ctx, userID, sessionID, n, "reveal", func(lc *lessonCtx, st state, lang string) error {
		_, qs, err := target(lc, n, similar)
		if err != nil {
			return err
		}
		if qs.Answered || qs.Revealed {
			return errf(409, "already_answered", false, "This question is already answered.")
		}
		if len(qs.Hints) < tutor.MaxHints {
			return errf(409, "hints_first", false, "Try both hints before giving up.")
		}
		qs.Revealed = true
		return nil
	})
}

// SimilarStep adds one similar question (step "3b") after a wrong answer.
func (s *Service) SimilarStep(ctx context.Context, userID, sessionID string, n int) (SessionView, error) {
	return s.stepAction(ctx, userID, sessionID, n, "similar", func(lc *lessonCtx, st state, lang string) error {
		q := lc.plan.Steps[n].Question
		ss := lc.state(n)
		if q == nil {
			return errf(409, "not_a_question", false, "This step has no question.")
		}
		if !(ss.Answered && !ss.Correct) && !ss.Revealed {
			return errf(409, "not_wrong", false, "A similar question is offered after a wrong answer.")
		}
		if ss.Similar != nil {
			return errf(409, "similar_used", false, "There's already a similar question for this step.")
		}
		objs := map[string]bool{q.ObjectiveID: true}
		var got *tutor.Question
		if s.checkQuota(ctx, userID) == nil {
			in := ai.SimilarInput{LessonInput: s.lessonInput(ctx, st, lc.node, lang), Question: *q}
			for attempt := 0; attempt < 2 && got == nil; attempt++ {
				res, err := s.AI.SimilarQuestion(ctx, in)
				s.recordUsage(ctx, userID, "similar", res.Usage)
				if err != nil {
					slog.Warn("similar call failed", "err", err)
					break
				}
				var nq tutor.Question
				if json.Unmarshal(res.Raw, &nq) == nil {
					if v, err := tutor.ValidateQuestion(nq, objs); err == nil && v.Prompt != q.Prompt {
						got = &v
					}
				}
			}
		}
		if got == nil {
			// Offline fallback: another practice prompt for the same objective.
			prompts := append(append([]curriculum.PracticePrompt{}, lc.lesson.GuidedPractice...), lc.lesson.IndependentPrompt...)
			for i, p := range prompts {
				if p.ObjectiveID == q.ObjectiveID && p.Prompt != q.Prompt {
					if v, ok := tutor.QuestionFromPrompt(p, prompts, i+1); ok {
						got = &v
						break
					}
				}
			}
		}
		if got == nil {
			return errf(502, "similar_unavailable", true, "Couldn't make a similar question right now. Please try again or continue.")
		}
		ss.Similar = &similarState{Question: *got}
		return nil
	})
}

type AskInput struct {
	Content string `json:"content"`
}

// AskStep answers a learner's question inside a step, within the lesson's scope.
func (s *Service) AskStep(ctx context.Context, userID, sessionID string, n int, in AskInput) (SessionView, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return SessionView{}, errf(400, "empty_message", false, "Message cannot be empty.")
	}
	if utf8.RuneCountInString(content) > MaxMessageRunes {
		return SessionView{}, errf(400, "message_too_long", false, "Message is too long (max %d characters).", MaxMessageRunes)
	}
	return s.stepAction(ctx, userID, sessionID, n, "ask", func(lc *lessonCtx, st state, lang string) error {
		ss := lc.state(n)
		if len(ss.Asks) >= maxAsksPerStep {
			return errf(429, "too_many_questions", false, "That's a lot of questions for one step — try moving on.")
		}
		if err := s.checkQuota(ctx, userID); err != nil {
			return err
		}
		ain := ai.AskInput{LessonInput: s.lessonInput(ctx, st, lc.node, lang), Step: lc.plan.Steps[n], Question: content}
		for _, a := range ss.Asks {
			ain.History = append(ain.History, ai.ChatMessage{Role: "user", Content: a.Question}, ai.ChatMessage{Role: "assistant", Content: a.Answer})
		}
		for _, other := range st.cur.Nodes {
			if other.ID != lc.node.ID {
				ain.OtherLessons = append(ain.OtherLessons, fmt.Sprintf("%s (%s)", st.cur.LocalNode(other.ID, lang).Title, other.Level))
			}
		}
		res, err := s.AI.Ask(ctx, ain)
		s.recordUsage(ctx, userID, "ask", res.Usage)
		if err != nil {
			slog.Warn("ask call failed", "provider", s.AI.Name(), "err", err)
			return errAIUnavailable
		}
		a, err := tutor.ParseAnswer(res.Raw)
		if err != nil {
			slog.Warn("ask reply rejected", "provider", s.AI.Name(), "err", err)
			return errAIUnavailable
		}
		ss.Asks = append(ss.Asks, askEntry{Question: content, Answer: a.Answer, Scope: a.Scope, OtherLesson: a.OtherLesson})
		if a.Remember != "" {
			if u, err := storage.GetUser(ctx, s.Store.Pool, userID); err == nil && u.MemoryEnabled {
				if err := storage.AddMemory(ctx, s.Store.Pool, userID, lc.cur.Slug, "personal", a.Remember, lc.node.ID); err != nil {
					slog.Warn("save memory", "err", err)
				}
			}
		}
		return nil
	})
}

// ---- after the checkpoint ----

type storedSummary struct {
	tutor.Summary
	FlashcardsAdded int    `json:"flashcardsAdded"`
	NextNodeID      string `json:"nextNodeId"`
}

// mistakes lists the practice questions answered wrong, with their answers.
func mistakes(lc *lessonCtx) (texts []string, cards []storage.NewFlashcard) {
	add := func(q tutor.Question, qs qState) {
		if !(qs.Answered && !qs.Correct) && !qs.Revealed {
			return
		}
		answer := q.Solution
		if q.Type == tutor.QuestionMCQ {
			answer = q.Options[q.Answer]
		}
		texts = append(texts, fmt.Sprintf("%s → %s", q.Prompt, answer))
		front := q.Prompt
		if utf8.RuneCountInString(front) > curriculum.FlashcardMax {
			front = string([]rune(front)[:curriculum.FlashcardMax-1]) + "…"
		}
		cards = append(cards, storage.NewFlashcard{Front: front, Back: answer})
	}
	for i, step := range lc.plan.Steps {
		if step.Question == nil || i >= len(lc.states) {
			continue
		}
		ss := lc.states[i]
		add(*step.Question, ss.qState)
		if ss.Similar != nil {
			add(ss.Similar.Question, ss.Similar.qState)
		}
	}
	return texts, cards
}

// finishLesson writes the lesson summary after a passed checkpoint, saves
// memory notes and mistake flashcards, and prepares the next lesson's opening.
// Failures here never undo the mastery that was just recorded.
func (s *Service) finishLesson(ctx context.Context, userID string, cur *curriculum.Curriculum, n *curriculum.Node, sess storage.Session, scores map[string]int) {
	lc, err := decodeLesson(cur, n, sess)
	if err != nil {
		slog.Warn("finish lesson: decode", "err", err)
		return
	}
	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		slog.Warn("finish lesson: load", "err", err)
		return
	}
	lang := cur.Lang(sess.Locale)
	mtexts, cards := mistakes(lc)
	out := storedSummary{}
	var next *curriculum.Node
	if _, nv := neighbours(st, n); nv != nil {
		next, _ = cur.Node(nv.ID)
	}
	in := ai.SummaryInput{LessonInput: s.lessonInput(ctx, st, n, lang), Passed: true, Scores: scores, Mistakes: mtexts}
	if next != nil {
		in.NextLesson = cur.LocalNode(next.ID, lang).Title
		out.NextNodeID = next.ID
	}
	got := false
	if s.checkQuota(ctx, userID) == nil {
		res, err := s.AI.Summarize(ctx, in)
		s.recordUsage(ctx, userID, "summary", res.Usage)
		if err == nil {
			if sum, err := tutor.ParseSummary(res.Raw); err == nil {
				out.Summary, got = sum, true
			}
		}
		if !got {
			slog.Warn("lesson summary unavailable; using scores", "node", n.ID, "err", err)
		}
	}
	if !got {
		out.Mastered, out.Review, out.Memory = []string{}, []string{}, []string{}
		for _, o := range cur.LocalNode(n.ID, lang).Objectives {
			if scores[o.ID] >= n.MasteryThreshold {
				out.Mastered = append(out.Mastered, o.Text)
			} else {
				out.Review = append(out.Review, o.Text)
			}
		}
	}
	if u, err := storage.GetUser(ctx, s.Store.Pool, userID); err == nil && u.MemoryEnabled {
		for _, m := range out.Memory {
			if err := storage.AddMemory(ctx, s.Store.Pool, userID, cur.Slug, "pattern", m, n.ID); err != nil {
				slog.Warn("save memory", "err", err)
			}
		}
	} else {
		out.Memory = []string{}
	}
	if len(cards) > 0 {
		if total, _, err := storage.CountFlashcards(ctx, s.Store.Pool, userID); err == nil && total+len(cards) <= maxCardsPerUser {
			if added, err := storage.InsertFlashcards(ctx, s.Store.Pool, userID, cur.Slug, n.ID, "mistake", cards); err == nil {
				out.FlashcardsAdded = added
			} else {
				slog.Warn("save mistake flashcards", "err", err)
			}
		}
	}
	raw, err := json.Marshal(out)
	if err == nil {
		err = storage.SetSessionSummary(ctx, s.Store.Pool, sess.ID, raw)
	}
	if err != nil {
		slog.Warn("save lesson summary", "err", err)
	}
}

var errStepsFirst = errf(409, "finish_steps_first", false, "Finish the lesson's steps before the checkpoint.")

// ErrIs reports whether err is an API error with the given code.
func ErrIs(err error, code string) bool {
	var le *Error
	return errors.As(err, &le) && le.Code == code
}
