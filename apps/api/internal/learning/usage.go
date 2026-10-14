package learning

import (
	"context"
	"log/slog"
	"time"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/storage"
)

// cacheTTL is how long shared AI output (lesson plans) is reused.
const cacheTTL = 7 * 24 * time.Hour

var (
	errQuota        = errf(429, "ai_quota", false, "You've used today's AI allowance. The lesson's steps you've opened and your flashcards still work; the tutor is back tomorrow.")
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
	if s.DailyTokens <= 0 || userID == "" {
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
	if u == (ai.Usage{}) || userID == "" {
		return
	}
	slog.Info("ai usage", "kind", kind, "provider", s.AI.Name(), "input", u.Input, "cached", u.Cached, "output", u.Output)
	if err := storage.AddUsage(ctx, s.Store.Pool, userID, s.day(), u.Input, u.Cached, u.Output); err != nil {
		slog.Warn("record ai usage", "err", err)
	}
}
