// Package storage persists users, login sessions, enrollments, progress,
// lesson sessions, messages and assessment attempts in PostgreSQL via pgx.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fluentpath/api/migrations"
)

var (
	ErrNotFound = errors.New("not found")
	// ErrConflict means the row changed since it was read (optimistic check).
	ErrConflict = errors.New("conflict")
)

// DB is satisfied by both *pgxpool.Pool and pgx.Tx.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	Pool *pgxpool.Pool
}

// Open connects with retries until ctx expires, so the API can start before
// the database is ready (e.g. under docker compose).
func Open(ctx context.Context, url string, searchPath string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if searchPath != "" {
		cfg.ConnConfig.RuntimeParams["search_path"] = searchPath
	}
	cfg.MaxConns = 10
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	var pool *pgxpool.Pool
	for attempt := 1; ; attempt++ {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return &Store{Pool: pool}, nil
			}
			pool.Close()
		}
		slog.Warn("database not ready, retrying", "attempt", attempt, "err", err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to database: %w", err)
		case <-time.After(time.Duration(min(attempt, 5)) * time.Second):
		}
	}
}

func (s *Store) Close() { s.Pool.Close() }

// Migrate applies embedded migrations in filename order under an advisory
// lock so concurrent API instances don't race.
func (s *Store) Migrate(ctx context.Context) error { return s.MigrateUpTo(ctx, "") }

// MigrateUpTo applies migrations up to and including `last` ("" = all).
func (s *Store) MigrateUpTo(ctx context.Context, last string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(724100)`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
			return err
		}
		names, err := fs.Glob(migrations.FS, "*.sql")
		if err != nil {
			return err
		}
		sort.Strings(names)
		for _, name := range names {
			if last != "" && name > last {
				break
			}
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			sqlBytes, err := migrations.FS.ReadFile(name)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
				return fmt.Errorf("migration %s: %w", name, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
				return err
			}
			slog.Info("applied migration", "version", name)
		}
		return nil
	})
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ---- users ----

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	DisplayName    string    `json:"displayName"`
	NativeLanguage string    `json:"nativeLanguage"`
	Locale         string    `json:"locale"` // UI language: en | id
	CreatedAt      time.Time `json:"createdAt"`
	PasswordHash   string    `json:"-"`
}

const userCols = `id, COALESCE(email, ''), display_name, native_language, locale, created_at, COALESCE(password_hash, '')`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.NativeLanguage, &u.Locale, &u.CreatedAt, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// CreateUser returns ErrConflict when the email is already registered.
func CreateUser(ctx context.Context, db DB, email, passwordHash, displayName, locale string) (User, error) {
	if locale == "" {
		locale = "en"
	}
	u, err := scanUser(db.QueryRow(ctx, `INSERT INTO users (id, email, password_hash, display_name, locale)
		VALUES (gen_random_uuid(), $1, $2, $3, $4) RETURNING `+userCols, email, passwordHash, displayName, locale))
	if isUniqueViolation(err) {
		return u, ErrConflict
	}
	return u, err
}

func GetUser(ctx context.Context, db DB, id string) (User, error) {
	return scanUser(db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
}

func GetUserByEmail(ctx context.Context, db DB, email string) (User, error) {
	return scanUser(db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email=$1`, email))
}

func SetUserLocale(ctx context.Context, db DB, id, locale string) (User, error) {
	return scanUser(db.QueryRow(ctx, `UPDATE users SET locale=$2 WHERE id=$1 RETURNING `+userCols, id, locale))
}

// ---- login sessions ----

func CreateAuthSession(ctx context.Context, db DB, userID, tokenHash, userAgent string, expires time.Time) error {
	if len(userAgent) > 300 {
		userAgent = userAgent[:300]
	}
	_, err := db.Exec(ctx, `INSERT INTO auth_sessions (user_id, token_hash, user_agent, expires_at) VALUES ($1,$2,$3,$4)`,
		userID, tokenHash, userAgent, expires)
	return err
}

// UserForToken returns the user behind an unexpired session token hash and
// slides the expiry forward (at most once a minute, to limit writes).
func UserForToken(ctx context.Context, db DB, tokenHash string, ttl time.Duration) (User, error) {
	u, err := scanUser(db.QueryRow(ctx, `SELECT u.id, COALESCE(u.email, ''), u.display_name, u.native_language, u.locale, u.created_at, ''
		FROM auth_sessions a JOIN users u ON u.id = a.user_id
		WHERE a.token_hash=$1 AND a.expires_at > NOW()`, tokenHash))
	if err != nil {
		return u, err
	}
	_, err = db.Exec(ctx, `UPDATE auth_sessions SET last_seen_at=NOW(), expires_at=NOW() + $2::interval
		WHERE token_hash=$1 AND last_seen_at < NOW() - INTERVAL '1 minute'`, tokenHash, fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	return u, err
}

func DeleteAuthSession(ctx context.Context, db DB, tokenHash string) error {
	_, err := db.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash=$1`, tokenHash)
	return err
}

func DeleteExpiredAuthSessions(ctx context.Context, db DB) (int64, error) {
	tag, err := db.Exec(ctx, `DELETE FROM auth_sessions WHERE expires_at <= NOW()`)
	return tag.RowsAffected(), err
}

// ---- enrollments ----

type Enrollment struct {
	CourseSlug     string    `json:"courseSlug"`
	CurrentLevel   string    `json:"currentLevel"`
	TargetLevel    string    `json:"targetLevel"`
	EnrolledAt     time.Time `json:"enrolledAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

const enrollmentCols = `course_slug, current_level, target_level, enrolled_at, last_activity_at`

func scanEnrollment(row pgx.Row) (Enrollment, error) {
	var e Enrollment
	err := row.Scan(&e.CourseSlug, &e.CurrentLevel, &e.TargetLevel, &e.EnrolledAt, &e.LastActivityAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

func GetEnrollment(ctx context.Context, db DB, userID, course string) (Enrollment, error) {
	return scanEnrollment(db.QueryRow(ctx, `SELECT `+enrollmentCols+` FROM enrollments WHERE user_id=$1 AND course_slug=$2`, userID, course))
}

// EnsureEnrollment enrols the user if they aren't already (idempotent).
func EnsureEnrollment(ctx context.Context, db DB, userID, course, currentLevel, targetLevel string) (Enrollment, error) {
	if _, err := db.Exec(ctx, `INSERT INTO enrollments (user_id, course_slug, current_level, target_level)
		VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, userID, course, currentLevel, targetLevel); err != nil {
		return Enrollment{}, err
	}
	return GetEnrollment(ctx, db, userID, course)
}

func ListEnrollments(ctx context.Context, db DB, userID string) ([]Enrollment, error) {
	rows, err := db.Query(ctx, `SELECT `+enrollmentCols+` FROM enrollments WHERE user_id=$1 ORDER BY last_activity_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Enrollment, error) { return scanEnrollment(r) })
}

func SetEnrollmentLevel(ctx context.Context, db DB, userID, course, level string) error {
	_, err := db.Exec(ctx, `UPDATE enrollments SET current_level=$3 WHERE user_id=$1 AND course_slug=$2`, userID, course, level)
	return err
}

func TouchEnrollment(ctx context.Context, db DB, userID, course string) error {
	_, err := db.Exec(ctx, `UPDATE enrollments SET last_activity_at=NOW() WHERE user_id=$1 AND course_slug=$2`, userID, course)
	return err
}

// ---- node progress ----

type NodeProgress struct {
	NodeID       string
	Status       string
	MasteryScore *int
	MasteredAt   *time.Time
	UpdatedAt    time.Time
}

func ListProgress(ctx context.Context, db DB, userID, course string) ([]NodeProgress, error) {
	rows, err := db.Query(ctx, `SELECT node_id, status, mastery_score, mastered_at, updated_at
		FROM node_progress WHERE user_id=$1 AND course_slug=$2`, userID, course)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (NodeProgress, error) {
		var p NodeProgress
		var score *int16
		err := r.Scan(&p.NodeID, &p.Status, &score, &p.MasteredAt, &p.UpdatedAt)
		if score != nil {
			v := int(*score)
			p.MasteryScore = &v
		}
		return p, err
	})
}

// MarkInProgress records that a lesson was started, never downgrading mastery.
func MarkInProgress(ctx context.Context, db DB, userID, course, nodeID string) error {
	_, err := db.Exec(ctx, `INSERT INTO node_progress (user_id, course_slug, node_id, status) VALUES ($1, $2, $3, 'in_progress')
		ON CONFLICT (user_id, course_slug, node_id) DO UPDATE SET status='in_progress', updated_at=NOW()
		WHERE node_progress.status <> 'mastered'`, userID, course, nodeID)
	return err
}

// LockProgress takes a row lock on the user's progress for a node (creating
// the row if needed) so completion is serialized. Returns the current status.
func LockProgress(ctx context.Context, tx pgx.Tx, userID, course, nodeID string) (string, error) {
	if _, err := tx.Exec(ctx, `INSERT INTO node_progress (user_id, course_slug, node_id, status) VALUES ($1, $2, $3, 'in_progress')
		ON CONFLICT DO NOTHING`, userID, course, nodeID); err != nil {
		return "", err
	}
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM node_progress WHERE user_id=$1 AND course_slug=$2 AND node_id=$3 FOR UPDATE`,
		userID, course, nodeID).Scan(&status)
	return status, err
}

// MarkMastered sets mastery exactly once; returns false if it was already mastered.
func MarkMastered(ctx context.Context, tx pgx.Tx, userID, course, nodeID string, score int) (bool, error) {
	tag, err := tx.Exec(ctx, `UPDATE node_progress SET status='mastered', mastery_score=$4, mastered_at=NOW(), updated_at=NOW()
		WHERE user_id=$1 AND course_slug=$2 AND node_id=$3 AND status <> 'mastered'`, userID, course, nodeID, score)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ---- lesson sessions ----

type Session struct {
	ID         string     `json:"id"`
	UserID     string     `json:"-"`
	CourseSlug string     `json:"courseSlug"`
	NodeID     string     `json:"nodeId"`
	Locale     string     `json:"locale"` // language the lesson runs in
	Phase      string     `json:"phase"`
	PhaseTurns int        `json:"phaseTurns"`
	StartedAt  time.Time  `json:"startedAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	EndedAt    *time.Time `json:"endedAt"`
}

const sessionCols = `id, user_id, course_slug, node_id, locale, phase, phase_turns, started_at, updated_at, ended_at`

func scanSession(row pgx.Row) (Session, error) {
	var s Session
	err := row.Scan(&s.ID, &s.UserID, &s.CourseSlug, &s.NodeID, &s.Locale, &s.Phase, &s.PhaseTurns, &s.StartedAt, &s.UpdatedAt, &s.EndedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

func FindOpenSession(ctx context.Context, db DB, userID, course, nodeID string) (Session, error) {
	return scanSession(db.QueryRow(ctx, `SELECT `+sessionCols+` FROM learning_sessions
		WHERE user_id=$1 AND course_slug=$2 AND node_id=$3 AND ended_at IS NULL`, userID, course, nodeID))
}

// CreateSession inserts a new open session. Returns ErrConflict if another
// open session for the node was created concurrently.
func CreateSession(ctx context.Context, db DB, userID, course, nodeID, locale, phase string) (Session, error) {
	s, err := scanSession(db.QueryRow(ctx, `INSERT INTO learning_sessions (user_id, course_slug, node_id, locale, phase)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+sessionCols, userID, course, nodeID, locale, phase))
	if isUniqueViolation(err) {
		return s, ErrConflict
	}
	return s, err
}

// GetSession only returns sessions owned by userID; others are ErrNotFound.
func GetSession(ctx context.Context, db DB, id, userID string) (Session, error) {
	return scanSession(db.QueryRow(ctx, `SELECT `+sessionCols+` FROM learning_sessions WHERE id=$1 AND user_id=$2`, id, userID))
}

func LockSession(ctx context.Context, tx pgx.Tx, id, userID string) (Session, error) {
	return scanSession(tx.QueryRow(ctx, `SELECT `+sessionCols+` FROM learning_sessions WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, userID))
}

// AdvanceSession moves a session to a new phase/turn count, but only if it is
// still in the expected state (optimistic concurrency).
func AdvanceSession(ctx context.Context, db DB, id, fromPhase string, fromTurns int, toPhase string, toTurns int) error {
	tag, err := db.Exec(ctx, `UPDATE learning_sessions SET phase=$4, phase_turns=$5, updated_at=NOW()
		WHERE id=$1 AND phase=$2 AND phase_turns=$3 AND ended_at IS NULL`, id, fromPhase, fromTurns, toPhase, toTurns)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func EndSession(ctx context.Context, db DB, id, phase string) error {
	_, err := db.Exec(ctx, `UPDATE learning_sessions SET phase=$2, phase_turns=0, ended_at=NOW(), updated_at=NOW() WHERE id=$1`, id, phase)
	return err
}

func SetSessionPhase(ctx context.Context, db DB, id, phase string) error {
	_, err := db.Exec(ctx, `UPDATE learning_sessions SET phase=$2, phase_turns=0, updated_at=NOW() WHERE id=$1`, id, phase)
	return err
}

// OpenSessions lists the user's unfinished sessions across courses, newest first.
func OpenSessions(ctx context.Context, db DB, userID string) ([]Session, error) {
	rows, err := db.Query(ctx, `SELECT `+sessionCols+` FROM learning_sessions
		WHERE user_id=$1 AND ended_at IS NULL ORDER BY updated_at DESC LIMIT 20`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Session, error) { return scanSession(r) })
}

// ---- messages ----

type Message struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Phase     string          `json:"phase"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"createdAt"`
}

func AddMessage(ctx context.Context, db DB, sessionID, role, content, phase string, metadata any) (Message, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	meta, err := json.Marshal(metadata)
	if err != nil {
		return Message{}, err
	}
	var m Message
	err = db.QueryRow(ctx, `INSERT INTO session_messages (session_id, role, content, phase, metadata)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, role, content, phase, metadata, created_at`,
		sessionID, role, content, phase, meta).Scan(&m.ID, &m.Role, &m.Content, &m.Phase, &m.Metadata, &m.CreatedAt)
	return m, err
}

func ListMessages(ctx context.Context, db DB, sessionID string) ([]Message, error) {
	rows, err := db.Query(ctx, `SELECT id, role, content, phase, metadata, created_at
		FROM session_messages WHERE session_id=$1 ORDER BY seq`, sessionID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Message, error) {
		var m Message
		err := r.Scan(&m.ID, &m.Role, &m.Content, &m.Phase, &m.Metadata, &m.CreatedAt)
		return m, err
	})
}

// RecentMistakes returns the most frequent mistake tags the tutor observed for
// this user in a course recently, most common first.
func RecentMistakes(ctx context.Context, db DB, userID, course string, limit int) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT tag FROM (
			SELECT jsonb_array_elements_text(m.metadata->'observedMistakes') AS tag
			FROM session_messages m JOIN learning_sessions s ON s.id = m.session_id
			WHERE s.user_id=$1 AND s.course_slug=$2 AND m.role='assistant' AND m.metadata ? 'observedMistakes'
			ORDER BY m.created_at DESC LIMIT 200) t
		GROUP BY tag ORDER BY COUNT(*) DESC, tag LIMIT $3`, userID, course, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// ---- assessment attempts ----

type Attempt struct {
	ID              string          `json:"id"`
	SessionID       string          `json:"sessionId"`
	CourseSlug      string          `json:"courseSlug,omitempty"`
	NodeID          string          `json:"nodeId,omitempty"`
	Score           int             `json:"score"`
	Threshold       int             `json:"threshold"`
	Passed          bool            `json:"passed"`
	ObjectiveScores map[string]int  `json:"objectiveScores"`
	Evidence        []string        `json:"evidence"`
	Feedback        string          `json:"feedback"`
	Answers         json.RawMessage `json:"-"`
	CreatedAt       time.Time       `json:"createdAt"`
}

type NewAttempt struct {
	SessionID         string
	Score, Threshold  int
	Passed            bool
	ObjectiveScores   map[string]int
	Evidence          []string
	Feedback          string
	Answers           map[string]string
	EvaluatorResponse json.RawMessage
	Evaluator         string
}

func InsertAttempt(ctx context.Context, db DB, a NewAttempt) (Attempt, error) {
	scores, _ := json.Marshal(a.ObjectiveScores)
	evidence, _ := json.Marshal(a.Evidence)
	answers, _ := json.Marshal(a.Answers)
	raw := a.EvaluatorResponse
	if !json.Valid(raw) {
		raw, _ = json.Marshal(string(raw))
	}
	return scanAttempt(db.QueryRow(ctx, `INSERT INTO assessment_attempts
		(session_id, score, threshold, passed, objective_scores, evidence, feedback, answers, evaluator_response, evaluator)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING `+attemptCols, a.SessionID, a.Score, a.Threshold, a.Passed, scores, evidence, a.Feedback, answers, raw, a.Evaluator))
}

const attemptCols = `id, session_id, score, threshold, passed, objective_scores, evidence, feedback, answers, created_at`

func scanAttempt(row pgx.Row) (Attempt, error) {
	var a Attempt
	var score, threshold int16
	var scores, evidence []byte
	err := row.Scan(&a.ID, &a.SessionID, &score, &threshold, &a.Passed, &scores, &evidence, &a.Feedback, &a.Answers, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil {
		return a, err
	}
	a.Score, a.Threshold = int(score), int(threshold)
	if err := json.Unmarshal(scores, &a.ObjectiveScores); err != nil {
		return a, err
	}
	if err := json.Unmarshal(evidence, &a.Evidence); err != nil {
		return a, err
	}
	return a, nil
}

func LatestAttempt(ctx context.Context, db DB, sessionID string) (Attempt, error) {
	return scanAttempt(db.QueryRow(ctx, `SELECT `+attemptCols+` FROM assessment_attempts
		WHERE session_id=$1 ORDER BY created_at DESC LIMIT 1`, sessionID))
}

// RecentAttempts lists the user's latest attempts; course "" means all courses.
func RecentAttempts(ctx context.Context, db DB, userID, course string, limit int) ([]Attempt, error) {
	rows, err := db.Query(ctx, `SELECT a.id, a.session_id, s.course_slug, s.node_id, a.score, a.threshold, a.passed,
			a.objective_scores, a.evidence, a.feedback, a.answers, a.created_at
		FROM assessment_attempts a JOIN learning_sessions s ON s.id = a.session_id
		WHERE s.user_id=$1 AND ($2 = '' OR s.course_slug = $2) ORDER BY a.created_at DESC LIMIT $3`, userID, course, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Attempt, error) {
		var a Attempt
		var score, threshold int16
		var scores, evidence []byte
		if err := r.Scan(&a.ID, &a.SessionID, &a.CourseSlug, &a.NodeID, &score, &threshold, &a.Passed, &scores, &evidence, &a.Feedback, &a.Answers, &a.CreatedAt); err != nil {
			return a, err
		}
		a.Score, a.Threshold = int(score), int(threshold)
		_ = json.Unmarshal(scores, &a.ObjectiveScores)
		_ = json.Unmarshal(evidence, &a.Evidence)
		return a, nil
	})
}

// ---- activity ----

// WeeklyActivity counts the user's replies, attempts and new masteries in the
// last 7 days (all courses).
func WeeklyActivity(ctx context.Context, db DB, userID string) (messages, attempts, mastered int, err error) {
	err = db.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM session_messages m JOIN learning_sessions s ON s.id=m.session_id
			WHERE s.user_id=$1 AND m.role='user' AND m.created_at > NOW() - INTERVAL '7 days'),
		(SELECT COUNT(*) FROM assessment_attempts a JOIN learning_sessions s ON s.id=a.session_id
			WHERE s.user_id=$1 AND a.created_at > NOW() - INTERVAL '7 days'),
		(SELECT COUNT(*) FROM node_progress WHERE user_id=$1 AND mastered_at > NOW() - INTERVAL '7 days')`,
		userID).Scan(&messages, &attempts, &mastered)
	return
}

// DailyActivity returns learner actions (replies + attempts) per local day
// for the last `days` days, keyed "2006-01-02" in the given time zone.
func DailyActivity(ctx context.Context, db DB, userID, tz string, days int) (map[string]int, error) {
	rows, err := db.Query(ctx, `SELECT to_char((t.at AT TIME ZONE $2)::date, 'YYYY-MM-DD'), COUNT(*) FROM (
			SELECT m.created_at AS at FROM session_messages m JOIN learning_sessions s ON s.id=m.session_id
				WHERE s.user_id=$1 AND m.role='user' AND m.created_at > NOW() - make_interval(days => $3)
			UNION ALL
			SELECT a.created_at FROM assessment_attempts a JOIN learning_sessions s ON s.id=a.session_id
				WHERE s.user_id=$1 AND a.created_at > NOW() - make_interval(days => $3)
		) t GROUP BY 1`, userID, tz, days)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for rows.Next() {
		var day string
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			rows.Close()
			return nil, err
		}
		out[day] = n
	}
	return out, rows.Err()
}
