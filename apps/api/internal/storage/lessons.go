package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- step-based lesson state ----

// SetSessionPlan stores a new plan and starts it from the first step.
func SetSessionPlan(ctx context.Context, db DB, id string, plan json.RawMessage) error {
	_, err := db.Exec(ctx, `UPDATE learning_sessions SET plan=$2, step_index=0, step_state='[]', updated_at=NOW() WHERE id=$1`, id, plan)
	return err
}

// SaveSteps stores the current step and the per-step state.
func SaveSteps(ctx context.Context, db DB, id string, stepIndex int, state json.RawMessage) error {
	_, err := db.Exec(ctx, `UPDATE learning_sessions SET step_index=$2, step_state=$3, updated_at=NOW() WHERE id=$1`, id, stepIndex, state)
	return err
}

func SetSessionSummary(ctx context.Context, db DB, id string, summary json.RawMessage) error {
	_, err := db.Exec(ctx, `UPDATE learning_sessions SET summary=$2 WHERE id=$1`, id, summary)
	return err
}

// LatestSummary returns the most recent lesson summary in a course other
// than nodeID (used to open the next lesson where the last one left off).
func LatestSummary(ctx context.Context, db DB, userID, course, nodeID string) (string, json.RawMessage, error) {
	var node string
	var sum json.RawMessage
	err := db.QueryRow(ctx, `SELECT node_id, summary FROM learning_sessions
		WHERE user_id=$1 AND course_slug=$2 AND node_id<>$3 AND summary IS NOT NULL
		ORDER BY ended_at DESC NULLS LAST, updated_at DESC LIMIT 1`, userID, course, nodeID).Scan(&node, &sum)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, nil
	}
	return node, sum, err
}

// ---- learner memory ----

type MemoryItem struct {
	ID         string    `json:"id"`
	CourseSlug string    `json:"courseSlug"`
	Kind       string    `json:"kind"` // pattern | personal
	Text       string    `json:"text"`
	SourceNode string    `json:"sourceNode"`
	CreatedAt  time.Time `json:"createdAt"`
}

const memoryCols = `id, course_slug, kind, text, source_node, created_at`

func scanMemory(row pgx.Row) (MemoryItem, error) {
	var m MemoryItem
	err := row.Scan(&m.ID, &m.CourseSlug, &m.Kind, &m.Text, &m.SourceNode, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

func AddMemory(ctx context.Context, db DB, userID, course, kind, text, node string) error {
	// Skip exact duplicates so repeated lessons don't pile up the same note.
	_, err := db.Exec(ctx, `INSERT INTO learner_memory (user_id, course_slug, kind, text, source_node)
		SELECT $1,$2,$3,$4,$5 WHERE NOT EXISTS (SELECT 1 FROM learner_memory WHERE user_id=$1 AND lower(text)=lower($4))`,
		userID, course, kind, text, node)
	return err
}

func ListMemory(ctx context.Context, db DB, userID string, limit int) ([]MemoryItem, error) {
	rows, err := db.Query(ctx, `SELECT `+memoryCols+` FROM learner_memory WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (MemoryItem, error) { return scanMemory(r) })
}

func UpdateMemory(ctx context.Context, db DB, id, userID, text string) (MemoryItem, error) {
	return scanMemory(db.QueryRow(ctx, `UPDATE learner_memory SET text=$3 WHERE id=$1 AND user_id=$2 RETURNING `+memoryCols, id, userID, text))
}

func DeleteMemory(ctx context.Context, db DB, id, userID string) (bool, error) {
	tag, err := db.Exec(ctx, `DELETE FROM learner_memory WHERE id=$1 AND user_id=$2`, id, userID)
	return tag.RowsAffected() > 0, err
}

func DeleteAllMemory(ctx context.Context, db DB, userID string) error {
	_, err := db.Exec(ctx, `DELETE FROM learner_memory WHERE user_id=$1`, userID)
	return err
}

// ---- notes (saved artifacts) ----

type Note struct {
	ID         string          `json:"id"`
	CourseSlug string          `json:"courseSlug"`
	NodeID     string          `json:"nodeId"`
	Title      string          `json:"title"`
	Content    json.RawMessage `json:"content"`
	CreatedAt  time.Time       `json:"createdAt"`
}

const noteCols = `id, course_slug, node_id, title, content, created_at`

func scanNote(row pgx.Row) (Note, error) {
	var n Note
	err := row.Scan(&n.ID, &n.CourseSlug, &n.NodeID, &n.Title, &n.Content, &n.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return n, ErrNotFound
	}
	return n, err
}

// SaveNote stores a note; saving the same title for the same lesson again replaces it.
func SaveNote(ctx context.Context, db DB, userID, course, node, title string, content json.RawMessage) (Note, error) {
	return scanNote(db.QueryRow(ctx, `INSERT INTO notes (user_id, course_slug, node_id, title, content) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id, course_slug, node_id, title) DO UPDATE SET content=EXCLUDED.content, created_at=NOW()
		RETURNING `+noteCols, userID, course, node, title, content))
}

func ListNotes(ctx context.Context, db DB, userID string) ([]Note, error) {
	rows, err := db.Query(ctx, `SELECT `+noteCols+` FROM notes WHERE user_id=$1 ORDER BY created_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Note, error) { return scanNote(r) })
}

func DeleteNote(ctx context.Context, db DB, id, userID string) (bool, error) {
	tag, err := db.Exec(ctx, `DELETE FROM notes WHERE id=$1 AND user_id=$2`, id, userID)
	return tag.RowsAffected() > 0, err
}
