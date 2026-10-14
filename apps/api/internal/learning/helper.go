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
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

const (
	// Only the latest messages go to the AI; older ones are summarised by
	// the material in the cached prompt prefix anyway.
	historyMessages   = 8
	historyReplyRunes = 600
	cacheTTL          = 7 * 24 * time.Hour
	maxResources      = 6
)

// MessageInput is one learner request to the helper. Quick-action buttons
// send an intent with no text; typed messages are intent "ask".
type MessageInput struct {
	Content string `json:"content"`
	Intent  string `json:"intent"`
}

// ResourceItem is a real-world link or picture shown in the helper.
type ResourceItem struct {
	Kind    string `json:"kind"` // video | article | image
	Title   string `json:"title"`
	URL     string `json:"url"`
	Source  string `json:"source"`
	Note    string `json:"note,omitempty"`
	VideoID string `json:"videoId,omitempty"` // YouTube id, for embeds
	Thumb   string `json:"thumb,omitempty"`   // images
	Author  string `json:"author,omitempty"`  // images: credit line
	License string `json:"license,omitempty"` // images
	Curated bool   `json:"curated"`
}

var (
	errQuota        = errf(429, "ai_quota", false, "You've used today's AI helper allowance. The material, quick checks and flashcards still work; the helper is back tomorrow.")
	errLessonClosed = errf(409, "lesson_closed", false, "This lesson is complete. Start it again from the roadmap to keep practising.")
)

func (s *Service) day() time.Time {
	loc, err := time.LoadLocation(s.TimeZone)
	if err != nil || s.TimeZone == "" {
		loc = time.UTC
	}
	return time.Now().In(loc)
}

// checkQuota enforces the daily per-learner token budget. Cached input
// tokens cost a fraction, so they don't count.
func (s *Service) checkQuota(ctx context.Context, userID string) error {
	if s.DailyTokens <= 0 {
		return nil
	}
	u, err := storage.UsageOn(ctx, s.Store.Pool, userID, s.day())
	if err != nil {
		return err
	}
	if u.Input-u.Cached+u.Output >= int64(s.DailyTokens) {
		return errQuota
	}
	return nil
}

func (s *Service) recordUsage(ctx context.Context, userID, kind string, u ai.Usage) {
	if u == (ai.Usage{}) {
		return
	}
	slog.Info("ai usage", "kind", kind, "provider", s.AI.Name(), "input", u.Input, "cached", u.Cached, "output", u.Output)
	if err := storage.AddUsage(ctx, s.Store.Pool, userID, s.day(), u.Input, u.Cached, u.Output); err != nil {
		slog.Warn("record ai usage", "err", err)
	}
}

type msgMeta struct {
	Kind       string               `json:"kind,omitempty"`
	Intent     tutor.Intent         `json:"intent,omitempty"`
	Quiz       []tutor.QuizQuestion `json:"quiz,omitempty"`
	Flashcards []tutor.Card         `json:"flashcards,omitempty"`
	Resources  []ResourceItem       `json:"resources,omitempty"`
	Cached     bool                 `json:"cached,omitempty"`
}

// trimHistory keeps the last few turns and shrinks them: long replies are
// cut and quizzes/cards/links are replaced by a short marker.
func trimHistory(msgs []storage.Message) []ai.ChatMessage {
	var out []ai.ChatMessage
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		var meta msgMeta
		_ = json.Unmarshal(m.Metadata, &meta)
		if meta.Kind == "review_intro" || meta.Kind == "result" {
			continue // the weak objectives are passed separately
		}
		text := m.Content
		if m.Role == "assistant" && utf8.RuneCountInString(text) > historyReplyRunes {
			text = string([]rune(text)[:historyReplyRunes]) + "…"
		}
		switch {
		case len(meta.Quiz) > 0:
			text += fmt.Sprintf(" [quiz: %d questions]", len(meta.Quiz))
		case len(meta.Flashcards) > 0:
			text += fmt.Sprintf(" [flashcards: %d]", len(meta.Flashcards))
		case len(meta.Resources) > 0:
			text += fmt.Sprintf(" [%d links shown]", len(meta.Resources))
		}
		out = append(out, ai.ChatMessage{Role: m.Role, Content: text})
	}
	if len(out) > historyMessages {
		out = out[len(out)-historyMessages:]
	}
	return out
}

// SendMessage answers one learner request. Nothing is stored unless the
// helper's reply is valid, so a failed call can simply be retried.
func (s *Service) SendMessage(ctx context.Context, userID, sessionID string, in MessageInput) (SessionView, error) {
	intent := tutor.Intent(strings.TrimSpace(in.Intent))
	if intent == "" {
		intent = tutor.IntentAsk
	}
	if !intent.Valid() {
		return SessionView{}, errf(400, "invalid_intent", false, "Unknown request type %q.", in.Intent)
	}
	content := strings.TrimSpace(in.Content)
	if content == "" && intent == tutor.IntentAsk {
		return SessionView{}, errf(400, "empty_message", false, "Message cannot be empty.")
	}
	if utf8.RuneCountInString(content) > MaxMessageRunes {
		return SessionView{}, errf(400, "message_too_long", false, "Message is too long (max %d characters).", MaxMessageRunes)
	}
	sess, err := s.getOwnSession(ctx, userID, sessionID)
	if err != nil {
		return SessionView{}, err
	}
	if sess.EndedAt != nil {
		return SessionView{}, errLessonClosed
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
	lang := st.lang
	// A quick-action button with no text: generic and cacheable.
	generic := content == ""
	if generic {
		content = tr(lang, "intent_"+string(intent))
	}
	history, err := storage.ListMessages(ctx, s.Store.Pool, sess.ID)
	if err != nil {
		return SessionView{}, err
	}

	// Generic requests share a cache across learners. The round number lets
	// a learner ask for a second, different quiz in the same lesson.
	round := 0
	for _, m := range history {
		var meta msgMeta
		if m.Role == "user" && json.Unmarshal(m.Metadata, &meta) == nil && meta.Intent == intent {
			round++
		}
	}
	key := fmt.Sprintf("helper:v1:%s:%s:%s:%s:%s:%s:%d", cur.Slug, cur.Version, n.ID, lang, intent, st.profile[curriculum.GoalQuestion], round)

	var reply tutor.Reply
	cached := false
	if generic {
		if raw, ok, err := storage.GetCache(ctx, s.Store.Pool, key); err != nil {
			return SessionView{}, err
		} else if ok {
			if reply, err = tutor.ParseReply(raw); err == nil {
				cached = true
			}
		}
	}
	if !cached {
		if err := s.checkQuota(ctx, userID); err != nil {
			return SessionView{}, err
		}
		tin := ai.TutorInput{
			Policy: cur.Tutor.Policy, Locale: lang, LanguagePolicy: cur.Tutor.LanguagePolicy[lang],
			CurrentLevel: st.enrollment.CurrentLevel, TargetLevel: st.enrollment.TargetLevel,
			Node: cur.LocalNode(n.ID, lang), Lesson: lesson,
			Intent: intent, CacheKey: fmt.Sprintf("%s:%s:%s", cur.Slug, n.ID, lang),
		}
		tin.LearnerProfile, tin.LearnerNotes = cur.TutorHints(st.profile)
		tin.WeakObjectives, err = s.weakObjectives(ctx, sess, cur, n, lang)
		if err != nil {
			return SessionView{}, err
		}
		tin.Messages = append(trimHistory(history), ai.ChatMessage{Role: "user", Content: content})
		res, err := s.AI.TutorReply(ctx, tin)
		if err != nil {
			slog.Warn("tutor call failed", "provider", s.AI.Name(), "session", sess.ID, "err", err)
			return SessionView{}, errAIUnavailable
		}
		s.recordUsage(ctx, userID, "helper", res.Usage)
		if reply, err = tutor.ParseReply(res.Raw); err != nil {
			slog.Warn("tutor reply rejected", "provider", s.AI.Name(), "session", sess.ID, "err", err)
			return SessionView{}, errAIUnavailable
		}
		if generic {
			if raw, err := json.Marshal(reply); err == nil {
				if err := storage.PutCache(ctx, s.Store.Pool, key, raw, cacheTTL); err != nil {
					slog.Warn("cache helper reply", "err", err)
				}
			}
		}
	}

	meta := msgMeta{Intent: intent, Quiz: reply.Quiz, Flashcards: reply.Flashcards, Cached: cached}
	kind, query := intent.SearchKind(), reply.Search.Query
	if reply.Search.Kind != "" {
		kind = reply.Search.Kind
	}
	if kind != "" {
		if query == "" {
			query = n.Title
		}
		meta.Resources = s.findResources(ctx, userID, cur, n, kind, query, lang)
	}

	err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		if _, err := storage.AddMessage(ctx, tx, sess.ID, "user", content, sess.Phase, msgMeta{Intent: intent}); err != nil {
			return err
		}
		if _, err := storage.AddMessage(ctx, tx, sess.ID, "assistant", reply.Message, sess.Phase, meta); err != nil {
			return err
		}
		return storage.TouchEnrollment(ctx, tx, userID, cur.Slug)
	})
	if err != nil {
		return SessionView{}, err
	}
	return s.GetSession(ctx, userID, sess.ID)
}

// weakObjectives lists the objectives the learner's last failed final quiz
// in this session scored low on.
func (s *Service) weakObjectives(ctx context.Context, sess storage.Session, cur *curriculum.Curriculum, n *curriculum.Node, lang string) ([]string, error) {
	a, err := storage.LatestAttempt(ctx, s.Store.Pool, sess.ID)
	if errors.Is(err, storage.ErrNotFound) || (err == nil && a.Passed) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, o := range cur.LocalNode(n.ID, lang).Objectives {
		if a.ObjectiveScores[o.ID] < a.Threshold {
			out = append(out, o.Text)
		}
	}
	return out, nil
}

// findResources returns curated links of a kind first, then fresh ones:
// pictures from Wikimedia Commons, videos/articles from the provider's web
// search. Search results are cached; failures fall back to curated links.
func (s *Service) findResources(ctx context.Context, userID string, cur *curriculum.Curriculum, n *curriculum.Node, kind, query, lang string) []ResourceItem {
	var out []ResourceItem
	seen := map[string]bool{}
	add := func(r ResourceItem) {
		if !seen[r.URL] && len(out) < maxResources {
			seen[r.URL] = true
			if r.Kind == tutor.KindVideo {
				r.VideoID, _ = curriculum.YouTubeID(r.URL)
			}
			out = append(out, r)
		}
	}
	if m := cur.MaterialIn(n.ID, lang); m != nil {
		for _, r := range m.Resources {
			if r.Kind == kind {
				add(ResourceItem{Kind: r.Kind, Title: r.Title, URL: r.URL, Source: r.Source, Note: r.Note, Curated: true})
			}
		}
	}

	norm := strings.Join(strings.Fields(strings.ToLower(query)), " ")
	key := fmt.Sprintf("search:v1:%s:%s", kind, norm)
	var found []ResourceItem
	if raw, ok, err := storage.GetCache(ctx, s.Store.Pool, key); err == nil && ok && json.Unmarshal(raw, &found) == nil {
		for _, r := range found {
			add(r)
		}
		return out
	}
	switch kind {
	case tutor.KindImage:
		if s.Commons == nil {
			return out
		}
		imgs, err := s.Commons.Search(ctx, query, 4)
		if err != nil {
			slog.Warn("commons search failed", "query", query, "err", err)
			return out
		}
		for _, im := range imgs {
			found = append(found, ResourceItem{Kind: kind, Title: im.Title, URL: im.PageURL, Source: "Wikimedia Commons",
				Thumb: im.Thumb, Author: im.Author, License: im.License})
		}
	default:
		if s.checkQuota(ctx, userID) != nil {
			return out
		}
		links, usage, err := s.AI.WebSearch(ctx, ai.SearchInput{Kind: kind, Query: query, LessonTitle: n.Title, Locale: lang})
		s.recordUsage(ctx, userID, "search", usage)
		if err != nil {
			if !errors.Is(err, ai.ErrNoSearch) {
				slog.Warn("web search failed", "query", query, "err", err)
			}
			return out
		}
		for _, l := range links {
			found = append(found, ResourceItem{Kind: kind, Title: l.Title, URL: l.URL, Source: l.Source})
		}
	}
	if raw, err := json.Marshal(found); err == nil && len(found) > 0 {
		if err := storage.PutCache(ctx, s.Store.Pool, key, raw, cacheTTL); err != nil {
			slog.Warn("cache search results", "err", err)
		}
	}
	for _, r := range found {
		add(r)
	}
	return out
}
