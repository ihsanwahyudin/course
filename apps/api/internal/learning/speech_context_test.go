package learning

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/storage"
)

func TestSpeechSendsOnlyTheRequestedText(t *testing.T) {
	s := &Service{AI: &audioMock{Mock: &ai.Mock{}}}
	for _, text := range []string{"I", "A", "I, as in ink.", "I am a teacher."} {
		b, err := s.say(context.Background(), text)
		if err != nil || string(b) != "mp3:"+text {
			t.Fatalf("speech must not inject an alphabet example: %s, %v", b, err)
		}
	}
}

func TestPronounAndAlphabetUseSeparateExistingRecordings(t *testing.T) {
	am := &audioMock{Mock: &ai.Mock{}}
	s := newService(t, am)
	s.LiveSpeech = true
	s.SpeechCacheDir = t.TempDir()
	cur, _ := s.Catalog.Course(course)
	// "eye" is the same pronunciation as I; its existing standalone recording
	// repairs the old bundle without making a new TTS call.
	want, err := os.ReadFile(cur.AudioPath("eye"))
	if err != nil {
		t.Fatal(err)
	}
	literal, err := s.Speech(context.Background(), learnerID, "I")
	if err != nil || !bytes.Equal(literal, want) {
		t.Fatal("I did not use the standalone recording", err)
	}
	letter, err := s.Speech(context.Background(), learnerID, "I, as in ink.")
	if err != nil || len(letter) == 0 || bytes.Equal(literal, letter) || am.speech.Load() != 0 {
		t.Fatal("alphabet audio must remain separate, with no provider call", err)
	}
	// After removing bundles, a legacy disk cache must not replay the suffix.
	withoutRecordedAudio(t, s)
	h := sha256.Sum256([]byte(am.Name() + "\x00I"))
	legacy := filepath.Join(s.SpeechCacheDir, hex.EncodeToString(h[:])+".mp3")
	if err := os.WriteFile(legacy, []byte("mp3:I, as in ink."), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := s.Speech(context.Background(), learnerID, "I")
	if err != nil || string(b) != "mp3:I" || am.speech.Load() != 1 {
		t.Fatalf("legacy expanded-letter cache was reused: %s, %v", b, err)
	}
}

func TestExplicitAlphabetAudioRefreshKeepsSessionAndChat(t *testing.T) {
	s := newService(t, &ai.Mock{})
	ctx := WithLang(context.Background(), "id")
	sess, err := s.StartSession(ctx, learnerID, course, "a0-alphabet-sounds")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AskStep(ctx, learnerID, sess.ID, 0, AskInput{Content: "Kenapa I dibaca ai?"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.NextStep(ctx, learnerID, sess.ID, 0); err != nil {
		t.Fatal(err)
	}
	old := planOf(t, s, sess.ID)
	for i := range old.Steps {
		for j := range old.Steps[i].Pairs {
			old.Steps[i].Pairs[j].SpeechText = ""
		}
	}
	raw, _ := json.Marshal(old)
	if err := storage.SetSessionLanguage(ctx, s.Store.Pool, sess.ID, "id", raw); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSession(ctx, learnerID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != sess.ID || got.Lesson.StepIndex != 1 || len(got.Lesson.Steps[0].Asks) != 1 || got.Lesson.Steps[0].Pairs[8].SpokenText() != "I, as in ink." {
		t.Fatalf("audio-only update: session=%s step=%d asks=%d I=%q", got.ID, got.Lesson.StepIndex, len(got.Lesson.Steps[0].Asks), got.Lesson.Steps[0].Pairs[8].SpokenText())
	}
}
