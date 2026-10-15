package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- checkpoints ----

// Checkpoint is one attempt's questions, answer keys included (server only).
type Checkpoint struct {
	ID          string
	SessionID   string
	Practice    bool
	Locale      string
	Source      string // provider name, or "authored" for the fallback
	Items       json.RawMessage
	CreatedAt   time.Time
	SubmittedAt *time.Time
}

const checkpointCols = `id, session_id, practice, locale, source, items, created_at, submitted_at`

func scanCheckpoint(row pgx.Row) (Checkpoint, error) {
	var c Checkpoint
	err := row.Scan(&c.ID, &c.SessionID, &c.Practice, &c.Locale, &c.Source, &c.Items, &c.CreatedAt, &c.SubmittedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// CreateCheckpoint stores a new open checkpoint. Returns ErrConflict when the
// session already has an open one.
func CreateCheckpoint(ctx context.Context, db DB, sessionID string, practice bool, locale, source string, items json.RawMessage) (Checkpoint, error) {
	c, err := scanCheckpoint(db.QueryRow(ctx, `INSERT INTO checkpoints (session_id, practice, locale, source, items)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+checkpointCols, sessionID, practice, locale, source, items))
	if isUniqueViolation(err) {
		return c, ErrConflict
	}
	return c, err
}

// OpenCheckpoint returns the session's unsubmitted checkpoint.
func OpenCheckpoint(ctx context.Context, db DB, sessionID string) (Checkpoint, error) {
	return scanCheckpoint(db.QueryRow(ctx, `SELECT `+checkpointCols+` FROM checkpoints
		WHERE session_id=$1 AND submitted_at IS NULL`, sessionID))
}

// GetCheckpoint returns one of the session's checkpoints.
func GetCheckpoint(ctx context.Context, db DB, sessionID, id string) (Checkpoint, error) {
	return scanCheckpoint(db.QueryRow(ctx, `SELECT `+checkpointCols+` FROM checkpoints WHERE id=$1 AND session_id=$2`, id, sessionID))
}

func LockCheckpoint(ctx context.Context, tx pgx.Tx, sessionID, id string) (Checkpoint, error) {
	return scanCheckpoint(tx.QueryRow(ctx, `SELECT `+checkpointCols+` FROM checkpoints WHERE id=$1 AND session_id=$2 FOR UPDATE`, id, sessionID))
}

func SubmitCheckpoint(ctx context.Context, db DB, id string) error {
	_, err := db.Exec(ctx, `UPDATE checkpoints SET submitted_at=NOW() WHERE id=$1`, id)
	return err
}

// EarlierCheckpoints returns the questions of the learner's latest
// checkpoints for a lesson (any session), newest first.
func EarlierCheckpoints(ctx context.Context, db DB, userID, course, nodeID string, limit int) ([]json.RawMessage, error) {
	rows, err := db.Query(ctx, `SELECT c.items FROM checkpoints c JOIN learning_sessions s ON s.id = c.session_id
		WHERE s.user_id=$1 AND s.course_slug=$2 AND s.node_id=$3 ORDER BY c.created_at DESC LIMIT $4`, userID, course, nodeID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (json.RawMessage, error) {
		var items json.RawMessage
		err := r.Scan(&items)
		return items, err
	})
}

// LatestSession returns the learner's most recent session for a lesson in a
// phase (a mastered lesson reopens on the session that mastered it).
func LatestSession(ctx context.Context, db DB, userID, course, nodeID, phase string) (Session, error) {
	return scanSession(db.QueryRow(ctx, `SELECT `+sessionCols+` FROM learning_sessions
		WHERE user_id=$1 AND course_slug=$2 AND node_id=$3 AND phase=$4 ORDER BY started_at DESC LIMIT 1`, userID, course, nodeID, phase))
}

// NodeAttempts is the learner's checkpoint history for a lesson (any
// session), newest first, without the per-answer review.
func NodeAttempts(ctx context.Context, db DB, userID, course, nodeID string, limit int) ([]Attempt, error) {
	rows, err := db.Query(ctx, `SELECT `+attemptColsA+`, s.course_slug, s.node_id
		FROM assessment_attempts a JOIN learning_sessions s ON s.id = a.session_id
		WHERE s.user_id=$1 AND s.course_slug=$2 AND s.node_id=$3 ORDER BY a.created_at DESC LIMIT $4`, userID, course, nodeID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Attempt, error) {
		a, err := scanAttemptRow(r, true)
		a.Review = nil
		return a, err
	})
}

// GetAttempt returns one of the user's attempts with its review.
func GetAttempt(ctx context.Context, db DB, userID, id string) (Attempt, error) {
	rows, err := db.Query(ctx, `SELECT `+attemptColsA+`, s.course_slug, s.node_id
		FROM assessment_attempts a JOIN learning_sessions s ON s.id = a.session_id WHERE a.id=$1 AND s.user_id=$2`, id, userID)
	if err != nil {
		return Attempt{}, err
	}
	a, err := pgx.CollectExactlyOneRow(rows, func(r pgx.CollectableRow) (Attempt, error) { return scanAttemptRow(r, true) })
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}
