package learning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/tutor"
)

func TestLegacyLowercaseAnswerCannotBePenalised(t *testing.T) {
	for _, keyWrong := range []bool{false, true} {
		name := "ordinary grade"
		if keyWrong {
			name = "incorrect keyWrong claim"
		}
		t.Run(name, func(t *testing.T) {
			mock := &ai.Mock{CheckpointFunc: func(ai.CheckpointInput) (json.RawMessage, error) { return aiCheckpoint(1), nil }}
			inner := &ai.Mock{}
			mock.GradeFunc = func(in ai.GradeInput) (json.RawMessage, error) {
				if in.Fixed["q4"] != 100 {
					t.Fatal("a and A must be a server-confirmed exact match")
				}
				r, err := inner.GradeCheckpoint(context.Background(), in)
				if err != nil {
					return nil, err
				}
				var out map[string]any
				if err := json.Unmarshal(r.Raw, &out); err != nil {
					return nil, err
				}
				for _, value := range out["items"].([]any) {
					row := value.(map[string]any)
					if row["id"] == "q4" {
						row["score"] = 0
						row["keyWrong"] = keyWrong
						row["correctAnswer"] = "A"
						row["explanation"] = "Untuk tugas ini diminta huruf besar A."
					}
				}
				out["feedback"] = "Tulis huruf besar saat diminta satu huruf."
				out["toReview"] = []string{"Gunakan huruf besar A"}
				return json.Marshal(out)
			}
			s := newService(t, mock)
			ctx := WithLang(context.Background(), "id")
			sess := startDone(t, s, ctx, "a0-alphabet-sounds")
			var raw []byte
			if err := s.Store.Pool.QueryRow(ctx, "SELECT items FROM checkpoints WHERE id=$1", sess.Checkpoint.ID).Scan(&raw); err != nil {
				t.Fatal(err)
			}

			var items []tutor.CheckItem
			if err := json.Unmarshal(raw, &items); err != nil {
				t.Fatal(err)
			}
			// Replay an already stored checkpoint, not a new draft: it must remain
			// readable and its old answers must get the corrected grading policy.
			items[3].Prompt = "Tuliskan 1 huruf yang benar. Jawab dalam bahasa Inggris."
			items[3].Listen = ""
			items[3].Accepted = []string{"A"}
			items[3].Rubric = "Full credit: A. No credit: other letters or more than one word."
			raw, _ = json.Marshal(items)
			if _, err := s.Store.Pool.Exec(ctx, "UPDATE checkpoints SET items=$1, locale='id' WHERE id=$2", raw, sess.Checkpoint.ID); err != nil {
				t.Fatal(err)
			}
			picks := answers(s, "a0-alphabet-sounds", true)
			picks["q4"] = "a"
			result, err := s.SubmitAssessment(ctx, learnerID, sess.ID, picks)
			if err != nil {
				t.Fatal(err)
			}
			var review tutor.Review
			if err = json.Unmarshal(result.Attempt.Review, &review); err != nil {
				t.Fatal(err)
			}
			q := review.Items[3]
			if q.Score != 100 || q.Verdict != tutor.VerdictCorrect || q.KeyFixed || q.CorrectAnswer != "A" || !strings.Contains(q.Explanation, "sama-sama diterima") {
				t.Fatalf("case-insensitive match must win with a consistent explanation: %+v", q)
			}
			if len(review.ToReview) != 0 || strings.Contains(result.Attempt.Feedback, "huruf besar") {
				t.Fatal("invented case requirement survived in overall feedback")
			}
			if !result.Passed || result.Attempt.Score != 100 {
				t.Fatalf("correct answers must pass: %+v", result.Attempt)
			}
		})
	}
}

func TestUnverifiedCheckpointUsesAuthoredFallback(t *testing.T) {
	cases := []string{"provider error", "invalid json", "empty", "missing question", "duplicate question", "unknown question", "missing verdict", "quota"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			mock := &ai.Mock{CheckpointFunc: func(ai.CheckpointInput) (json.RawMessage, error) { return aiCheckpoint(1), nil }}
			mock.VerifyFunc = func(in ai.VerifyInput) (json.RawMessage, error) {
				if name == "provider error" {
					return nil, errors.New("verifier unavailable")
				}
				if name == "invalid json" {
					return json.RawMessage(`{`), nil
				}
				rows := []map[string]any{}
				for _, q := range in.Items {
					rows = append(rows, map[string]any{"id": q.ID, "ok": true})
				}
				switch name {
				case "empty":
					rows = nil
				case "missing question":
					rows = rows[:len(rows)-1]
				case "duplicate question":
					rows[1]["id"] = rows[0]["id"]
				case "unknown question":
					rows[0]["id"] = "unknown"
				case "missing verdict":
					delete(rows[0], "ok")
				}
				return json.Marshal(map[string]any{"items": rows})
			}
			s := newService(t, mock)
			if name == "quota" {
				s.DailyTokens = 1
			}
			sess := startDone(t, s, context.Background(), "a0-alphabet-sounds")
			var source string
			if err := s.Store.Pool.QueryRow(context.Background(), "SELECT source FROM checkpoints WHERE id=$1", sess.Checkpoint.ID).Scan(&source); err != nil {
				t.Fatal(err)
			}
			if source != "authored" || mock.Calls.Checkpoint != 1 {
				t.Fatalf("unverified questions must fall back immediately: source=%s calls=%d", source, mock.Calls.Checkpoint)
			}
		})
	}
}

func TestContextlessLetterDraftIsRejectedBeforeAIReview(t *testing.T) {
	mock := &ai.Mock{CheckpointFunc: func(ai.CheckpointInput) (json.RawMessage, error) {
		var out struct {
			Items []tutor.DraftItem `json:"items"`
		}
		if err := json.Unmarshal(aiCheckpoint(1), &out); err != nil {
			return nil, err
		}
		out.Items[3].Prompt = "Tuliskan 1 huruf yang benar. Jawab dalam bahasa Inggris."
		out.Items[3].Listen = ""
		out.Items[3].Accepted = []string{"A"}
		return json.Marshal(out)
	}}
	s := newService(t, mock)
	sess := startDone(t, s, context.Background(), "a0-alphabet-sounds")
	var source string
	if err := s.Store.Pool.QueryRow(context.Background(), "SELECT source FROM checkpoints WHERE id=$1", sess.Checkpoint.ID).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if source != "authored" || mock.Calls.Verify != 0 {
		t.Fatalf("a bare answer instruction must not reach semantic review: source=%s verify=%d", source, mock.Calls.Verify)
	}
}
