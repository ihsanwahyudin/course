package learning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

func TestUntaughtSoundChoicesRetryBeforeVerification(t *testing.T) {
	for _, lang := range []string{"en", "id"} {
		for _, repair := range []bool{false, true} {
			name := lang + "/fallback"
			if repair {
				name = lang + "/repaired"
			}
			t.Run(name, func(t *testing.T) {
				mock := &ai.Mock{}
				s := newService(t, mock)
				ctx := WithLang(context.Background(), lang)
				alphabet := startDone(t, s, ctx, "a0-alphabet-sounds")
				if result, err := s.SubmitAssessment(ctx, learnerID, alphabet.ID, answers(s, "a0-alphabet-sounds", true)); err != nil || !result.Passed {
					t.Fatalf("unlock: %v", err)
				}
				mock.Calls.Checkpoint, mock.Calls.Verify = 0, 0
				written := 0
				mock.CheckpointFunc = func(in ai.CheckpointInput) (json.RawMessage, error) {
					written++
					if written == 2 && (len(in.Problems) == 0 || !strings.Contains(in.Problems[0], "taught vocabulary")) {
						t.Fatalf("retry lacks concrete vocabulary feedback: %v", in.Problems)
					}
					ds := tutor.AuthoredDrafts(in.Lesson, in.Locale)
					for i := range ds {
						ds[i].Prompt = "Checkpoint. " + ds[i].Prompt
					}
					if !repair || written == 1 {
						ds[0].Options = []string{"leave", "live", "life"}
						ds[0].CorrectOption, ds[0].Listen = "leave", "leave"
					}
					return json.Marshal(map[string]any{"items": ds})
				}
				sess := startDone(t, s, ctx, "a0-sounds-indonesian")
				var source string
				var raw []byte
				if err := s.Store.Pool.QueryRow(ctx, "SELECT source,items FROM checkpoints WHERE id=$1", sess.Checkpoint.ID).Scan(&source, &raw); err != nil {
					t.Fatal(err)
				}
				wantSource, wantVerify := "authored", 0
				if repair {
					wantSource, wantVerify = "mock", 1
				}
				if source != wantSource || mock.Calls.Checkpoint != 2 || mock.Calls.Verify != wantVerify {
					t.Fatalf("source=%s writes=%d verifier=%d", source, mock.Calls.Checkpoint, mock.Calls.Verify)
				}
				var items []tutor.CheckItem
				if err := json.Unmarshal(raw, &items); err != nil {
					t.Fatal(err)
				}
				cur, _ := s.Catalog.Course(course)
				l, _ := cur.Lesson("a0-sounds-indonesian")
				if err := tutor.ValidateRecognitionVocabulary(items, l.RecognitionVocabulary); err != nil {
					t.Fatal("stored checkpoint escaped vocabulary check", err)
				}
				result, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-sounds-indonesian", true))
				if err != nil || !result.Passed || result.Attempt.Score != 100 {
					t.Fatalf("valid checkpoint must unlock normally: %v, %+v", err, result.Attempt)
				}
			})
		}
	}
}

func TestLevelCheckpointUsesEachLessonsRecognitionBank(t *testing.T) {
	mock := &ai.Mock{}
	s := newService(t, mock)
	ctx := WithLang(context.Background(), "id")
	cur, _ := s.Catalog.Course(course)
	node, _ := cur.Node("a0-checkpoint")
	lessons, _ := cur.LevelLessons(node)
	if _, err := s.Enroll(ctx, learnerID, course); err != nil {
		t.Fatal(err)
	}
	if err := pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		for _, lesson := range lessons {
			if _, err := storage.LockProgress(ctx, tx, learnerID, course, lesson.ID); err != nil {
				return err
			}
			if _, err := storage.MarkMastered(ctx, tx, learnerID, course, lesson.ID, 100); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	written := 0
	mock.CheckpointFunc = func(in ai.CheckpointInput) (json.RawMessage, error) {
		written++
		if written == 2 && (len(in.Problems) == 0 || !strings.Contains(in.Problems[0], "taught vocabulary")) {
			t.Fatalf("level retry lacks vocabulary feedback: %v", in.Problems)
		}
		var drafts []tutor.DraftItem
		found := false
		for _, lesson := range in.LevelLessons {
			d := tutor.AuthoredDraft(lesson.Sample, "Checkpoint. "+lesson.SamplePrompt, lesson.SampleDisplayOptions)
			d.LessonID, d.ObjectiveID = lesson.ID, lesson.Objective
			if lesson.ID == "a0-sounds-indonesian" {
				found = true
				if len(lesson.RecognitionVocabulary) == 0 {
					t.Fatal("writer was not given the sound lesson's vocabulary bank")
				}
				d.Options = []string{"leave", "live", "life"}
				d.CorrectOption, d.Listen = "leave", "leave"
			}
			drafts = append(drafts, d)
		}
		if !found {
			t.Fatal("level checkpoint omitted sound lesson")
		}
		return json.Marshal(map[string]any{"items": drafts})
	}
	sess, err := s.StartSession(ctx, learnerID, course, node.ID)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := s.StartCheckpoint(ctx, learnerID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	var source string
	if err := s.Store.Pool.QueryRow(ctx, "SELECT source FROM checkpoints WHERE id=$1", cp.ID).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if source != "authored" || written != 2 || mock.Calls.Verify != 0 {
		t.Fatalf("invalid level vocabulary was not rejected before verification: %s, %d writes, %d verifications", source, written, mock.Calls.Verify)
	}
}
