package learning

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/storage"
)

const (
	maxCardsPerSave = 20
	maxCardsPerUser = 1000
	reviewBatch     = 50
)

type FlashcardInput struct {
	Course string `json:"course"`
	NodeID string `json:"nodeId"`
	Cards  []struct {
		Front string `json:"front"`
		Back  string `json:"back"`
	} `json:"cards"`
}

type SaveFlashcardsResult struct {
	Added int `json:"added"`
	Total int `json:"total"`
	Due   int `json:"due"`
}

// FlashcardView is a saved card with its lesson's name.
type FlashcardView struct {
	storage.Flashcard
	NodeTitle   string `json:"nodeTitle"`
	CourseTitle string `json:"courseTitle"`
}

type ReviewQueue struct {
	Cards []FlashcardView `json:"cards"`
	Due   int             `json:"due"`
	Total int             `json:"total"`
}

// SaveFlashcards stores cards from a lesson (its starter deck or the
// helper's), skipping ones the learner already has.
func (s *Service) SaveFlashcards(ctx context.Context, userID string, in FlashcardInput) (SaveFlashcardsResult, error) {
	cur, err := s.course(in.Course)
	if err != nil {
		return SaveFlashcardsResult{}, err
	}
	lesson, ok := cur.Lesson(in.NodeID)
	if !ok {
		return SaveFlashcardsResult{}, errf(404, "node_not_found", false, "No lesson %q.", in.NodeID)
	}
	if len(in.Cards) == 0 || len(in.Cards) > maxCardsPerSave {
		return SaveFlashcardsResult{}, errf(400, "invalid_flashcards", false, "Send between 1 and %d cards.", maxCardsPerSave)
	}
	starter := map[string]bool{}
	for _, f := range lesson.Flashcards {
		starter[f.Front+"\x00"+f.Back] = true
	}
	var cards []storage.NewFlashcard
	source := "lesson"
	for _, c := range in.Cards {
		front, back := strings.TrimSpace(c.Front), strings.TrimSpace(c.Back)
		if front == "" || back == "" || utf8.RuneCountInString(front) > curriculum.FlashcardMax || utf8.RuneCountInString(back) > curriculum.FlashcardMax {
			return SaveFlashcardsResult{}, errf(400, "invalid_flashcards", false, "Each side of a card needs 1–%d characters.", curriculum.FlashcardMax)
		}
		if !starter[front+"\x00"+back] {
			source = "ai"
		}
		cards = append(cards, storage.NewFlashcard{Front: front, Back: back})
	}
	var out SaveFlashcardsResult
	err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		// Serialise per user so two saves can't both slip under the limit.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('flashcards:' || $1))`, userID); err != nil {
			return err
		}
		total, _, err := storage.CountFlashcards(ctx, tx, userID)
		if err != nil {
			return err
		}
		if total+len(cards) > maxCardsPerUser {
			return errf(409, "flashcard_limit", false, "You can keep up to %d flashcards. Delete some to add more.", maxCardsPerUser)
		}
		if out.Added, err = storage.InsertFlashcards(ctx, tx, userID, cur.Slug, in.NodeID, source, cards); err != nil {
			return err
		}
		out.Total, out.Due, err = storage.CountFlashcards(ctx, tx, userID)
		return err
	})
	return out, err
}

// DueFlashcards returns the cards to review now ("" course = all courses).
func (s *Service) DueFlashcards(ctx context.Context, userID, course string) (ReviewQueue, error) {
	if course != "" {
		if _, err := s.course(course); err != nil {
			return ReviewQueue{}, err
		}
	}
	lang := langFrom(ctx)
	rows, err := storage.DueFlashcards(ctx, s.Store.Pool, userID, course, reviewBatch)
	if err != nil {
		return ReviewQueue{}, err
	}
	q := ReviewQueue{Cards: []FlashcardView{}}
	for _, f := range rows {
		v := FlashcardView{Flashcard: f}
		if cur, ok := s.Catalog.Course(f.CourseSlug); ok {
			v.CourseTitle = cur.Text(lang).Title
			if n := cur.LocalNode(f.NodeID, lang); n != nil {
				v.NodeTitle = n.Title
			}
		}
		q.Cards = append(q.Cards, v)
	}
	q.Total, q.Due, err = storage.CountFlashcards(ctx, s.Store.Pool, userID)
	return q, err
}

// Review grades.
const (
	GradeAgain = "again"
	GradeGood  = "good"
	GradeEasy  = "easy"
)

// schedule applies a simple SM-2-style rule: "again" brings the card back
// in ten minutes and resets it; "good" and "easy" grow the interval.
func schedule(f storage.Flashcard, grade string, now time.Time) storage.Flashcard {
	switch grade {
	case GradeAgain:
		f.Lapses++
		f.IntervalDays = 0
		f.DueAt = now.Add(10 * time.Minute)
		return f
	case GradeGood:
		if f.IntervalDays < 1 {
			f.IntervalDays = 1
		} else {
			f.IntervalDays = math.Round(f.IntervalDays*2.5*10) / 10
		}
	case GradeEasy:
		if f.IntervalDays < 1 {
			f.IntervalDays = 3
		} else {
			f.IntervalDays = math.Round(f.IntervalDays*3.5*10) / 10
		}
	}
	f.IntervalDays = math.Min(f.IntervalDays, 365)
	f.Reps++
	f.DueAt = now.Add(time.Duration(f.IntervalDays * 24 * float64(time.Hour)))
	return f
}

func (s *Service) ReviewFlashcard(ctx context.Context, userID, id, grade string) (storage.Flashcard, error) {
	if grade != GradeAgain && grade != GradeGood && grade != GradeEasy {
		return storage.Flashcard{}, errf(400, "invalid_grade", false, "Grade must be again, good or easy.")
	}
	var out storage.Flashcard
	err := pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		f, err := storage.LockFlashcard(ctx, tx, id, userID)
		if errors.Is(err, storage.ErrNotFound) {
			return errf(404, "flashcard_not_found", false, "Flashcard not found.")
		}
		if err != nil {
			return err
		}
		out = schedule(f, grade, time.Now())
		return storage.ScheduleFlashcard(ctx, tx, out)
	})
	return out, err
}

func (s *Service) DeleteFlashcard(ctx context.Context, userID, id string) error {
	ok, err := storage.DeleteFlashcard(ctx, s.Store.Pool, id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errf(404, "flashcard_not_found", false, "Flashcard not found.")
	}
	return nil
}
