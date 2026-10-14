package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- flashcards ----

type Flashcard struct {
	ID           string    `json:"id"`
	CourseSlug   string    `json:"courseSlug"`
	NodeID       string    `json:"nodeId"`
	Front        string    `json:"front"`
	Back         string    `json:"back"`
	Source       string    `json:"source"` // lesson | ai
	DueAt        time.Time `json:"dueAt"`
	IntervalDays float64   `json:"intervalDays"`
	Reps         int       `json:"reps"`
	Lapses       int       `json:"lapses"`
	CreatedAt    time.Time `json:"createdAt"`
}

const flashcardCols = `id, course_slug, node_id, front, back, source, due_at, interval_days, reps, lapses, created_at`

func scanFlashcard(row pgx.Row) (Flashcard, error) {
	var f Flashcard
	err := row.Scan(&f.ID, &f.CourseSlug, &f.NodeID, &f.Front, &f.Back, &f.Source, &f.DueAt, &f.IntervalDays, &f.Reps, &f.Lapses, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}

type NewFlashcard struct{ Front, Back string }

// InsertFlashcards saves cards due now; a card with the same front for the
// same lesson is skipped. Returns how many were added.
func InsertFlashcards(ctx context.Context, db DB, userID, course, nodeID, source string, cards []NewFlashcard) (int, error) {
	added := 0
	for _, c := range cards {
		tag, err := db.Exec(ctx, `INSERT INTO flashcards (user_id, course_slug, node_id, front, back, source)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, userID, course, nodeID, c.Front, c.Back, source)
		if err != nil {
			return added, err
		}
		added += int(tag.RowsAffected())
	}
	return added, nil
}

func CountFlashcards(ctx context.Context, db DB, userID string) (total, due int, err error) {
	err = db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE due_at <= NOW()) FROM flashcards WHERE user_id=$1`, userID).Scan(&total, &due)
	return
}

// DueFlashcards returns the cards due now, oldest due first ("" = every course).
func DueFlashcards(ctx context.Context, db DB, userID, course string, limit int) ([]Flashcard, error) {
	rows, err := db.Query(ctx, `SELECT `+flashcardCols+` FROM flashcards
		WHERE user_id=$1 AND due_at <= NOW() AND ($2 = '' OR course_slug = $2)
		ORDER BY due_at, created_at LIMIT $3`, userID, course, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Flashcard, error) { return scanFlashcard(r) })
}

func LockFlashcard(ctx context.Context, tx pgx.Tx, id, userID string) (Flashcard, error) {
	return scanFlashcard(tx.QueryRow(ctx, `SELECT `+flashcardCols+` FROM flashcards WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, userID))
}

func ScheduleFlashcard(ctx context.Context, db DB, f Flashcard) error {
	_, err := db.Exec(ctx, `UPDATE flashcards SET due_at=$2, interval_days=$3, reps=$4, lapses=$5 WHERE id=$1`,
		f.ID, f.DueAt, f.IntervalDays, f.Reps, f.Lapses)
	return err
}

func DeleteFlashcard(ctx context.Context, db DB, id, userID string) (bool, error) {
	tag, err := db.Exec(ctx, `DELETE FROM flashcards WHERE id=$1 AND user_id=$2`, id, userID)
	return tag.RowsAffected() > 0, err
}

// ---- AI cache and usage ----

// GetCache returns a cached AI result if it hasn't expired.
func GetCache(ctx context.Context, db DB, key string) (json.RawMessage, bool, error) {
	var v json.RawMessage
	err := db.QueryRow(ctx, `SELECT value FROM ai_cache WHERE key=$1 AND expires_at > NOW()`, key).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	return v, err == nil, err
}

func PutCache(ctx context.Context, db DB, key string, value json.RawMessage, ttl time.Duration) error {
	_, err := db.Exec(ctx, `INSERT INTO ai_cache (key, value, expires_at) VALUES ($1,$2,$3)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, expires_at=EXCLUDED.expires_at`, key, value, time.Now().Add(ttl))
	return err
}

func DeleteExpiredCache(ctx context.Context, db DB) (int64, error) {
	tag, err := db.Exec(ctx, `DELETE FROM ai_cache WHERE expires_at <= NOW()`)
	return tag.RowsAffected(), err
}

type TokenUsage struct {
	Input, Cached, Output int64
	Calls                 int
}

// AddUsage records one provider call's tokens against a learner's day.
func AddUsage(ctx context.Context, db DB, userID string, day time.Time, input, cached, output int) error {
	_, err := db.Exec(ctx, `INSERT INTO ai_usage (user_id, day, input_tokens, cached_tokens, output_tokens, calls)
		VALUES ($1,$2,$3,$4,$5,1)
		ON CONFLICT (user_id, day) DO UPDATE SET input_tokens = ai_usage.input_tokens + EXCLUDED.input_tokens,
			cached_tokens = ai_usage.cached_tokens + EXCLUDED.cached_tokens,
			output_tokens = ai_usage.output_tokens + EXCLUDED.output_tokens, calls = ai_usage.calls + 1`,
		userID, day.Format("2006-01-02"), input, cached, output)
	return err
}

func UsageOn(ctx context.Context, db DB, userID string, day time.Time) (TokenUsage, error) {
	var u TokenUsage
	err := db.QueryRow(ctx, `SELECT input_tokens, cached_tokens, output_tokens, calls FROM ai_usage WHERE user_id=$1 AND day=$2`,
		userID, day.Format("2006-01-02")).Scan(&u.Input, &u.Cached, &u.Output, &u.Calls)
	if errors.Is(err, pgx.ErrNoRows) {
		return TokenUsage{}, nil
	}
	return u, err
}
