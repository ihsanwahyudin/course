package learning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/tutor"
)

func TestCircularSoundQuestionRetriesBeforeReviewAndFallsBackSafely(t *testing.T) {
	for _, repair := range []bool{false, true} {
		name := "authored fallback"
		if repair {
			name = "repaired retry"
		}
		t.Run(name, func(t *testing.T) {
			mock := &ai.Mock{}
			s := newService(t, mock)
			ctx := WithLang(context.Background(), "id")
			// Unlock the reported lesson through the normal learning loop.
			alphabet := startDone(t, s, ctx, "a0-alphabet-sounds")
			if result, err := s.SubmitAssessment(ctx, learnerID, alphabet.ID, answers(s, "a0-alphabet-sounds", true)); err != nil || !result.Passed {
				t.Fatalf("unlock: %v", err)
			}
			mock.Calls.Checkpoint, mock.Calls.Verify = 0, 0
			written := 0
			mock.CheckpointFunc = func(in ai.CheckpointInput) (json.RawMessage, error) {
				written++
				if written == 2 && (len(in.Problems) == 0 || !strings.Contains(in.Problems[0], "must differ from reference word")) {
					t.Fatalf("retry lost the concrete problem: %v", in.Problems)
				}
				ds := tutor.AuthoredDrafts(in.Lesson, in.Locale)
				for i := range ds {
					ds[i].Prompt = "Soal latihan baru. " + ds[i].Prompt
				}
				q := &ds[0]
				q.Prompt = "Kata mana yang punya bunyi pendek /ɪ/ seperti pada 'ship'?"
				q.Listen = ""
				q.Options = []string{"ship", "seat", "food"}
				q.CorrectOption = "ship"
				q.Explanation = "Bunyi pendek /ɪ/ pada ship."
				if repair && written == 2 {
					q.Prompt = "Kata lain mana yang punya bunyi pendek /ɪ/ seperti pada 'ship'?"
					q.Options[0] = "sit"
					q.CorrectOption = "sit"
					q.Explanation = "Sit dan ship memakai /ɪ/ pendek."
				}
				return json.Marshal(map[string]any{"items": ds})
			}
			sess := startDone(t, s, ctx, "a0-sounds-indonesian")
			var source string
			var raw []byte
			if err := s.Store.Pool.QueryRow(ctx, "SELECT source,items FROM checkpoints WHERE id=$1", sess.Checkpoint.ID).Scan(&source, &raw); err != nil {
				t.Fatal(err)
			}
			wantSource := "authored"
			wantVerify := 0
			if repair {
				wantSource = "mock"
				wantVerify = 1
			}
			if source != wantSource || mock.Calls.Checkpoint != 2 || mock.Calls.Verify != wantVerify {
				t.Fatalf("source=%s writes=%d verify=%d", source, mock.Calls.Checkpoint, mock.Calls.Verify)
			}
			var items []tutor.CheckItem
			if err := json.Unmarshal(raw, &items); err != nil {
				t.Fatal(err)
			}
			if repair && (items[0].Key() != "sit" || strings.Contains(strings.Join(items[0].Options, " "), "ship")) {
				t.Fatalf("repaired question still circular: %+v", items[0])
			}
			// The resulting valid checkpoint must still grade correctly and unlock progress.
			result, err := s.SubmitAssessment(ctx, learnerID, sess.ID, answers(s, "a0-sounds-indonesian", true))
			if err != nil || !result.Passed || result.Attempt.Score != 100 {
				t.Fatalf("valid checkpoint should pass: %v, %+v", err, result.Attempt)
			}
		})
	}
}
