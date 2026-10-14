package learning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

// ---- learner memory ("Yang tutor ingat tentangmu") ----

// ProfileFact is an onboarding answer shown in the memory page; it's edited
// through the course's onboarding, not here.
type ProfileFact struct {
	Course      string `json:"course"`
	CourseTitle string `json:"courseTitle"`
	Question    string `json:"question"`
	Answer      string `json:"answer"`
}

// MemoryItemView is a memory note with the lesson it came from.
type MemoryItemView struct {
	storage.MemoryItem
	SourceTitle string `json:"sourceTitle"`
}

type MemoryView struct {
	Enabled bool             `json:"enabled"`
	Profile []ProfileFact    `json:"profile"`
	Items   []MemoryItemView `json:"items"`
}

func (s *Service) Memory(ctx context.Context, userID string) (MemoryView, error) {
	u, err := storage.GetUser(ctx, s.Store.Pool, userID)
	if err != nil {
		return MemoryView{}, err
	}
	v := MemoryView{Enabled: u.MemoryEnabled, Profile: []ProfileFact{}, Items: []MemoryItemView{}}
	lang := langFrom(ctx)
	enrollments, err := storage.ListEnrollments(ctx, s.Store.Pool, userID)
	if err != nil {
		return v, err
	}
	for _, e := range enrollments {
		cur, ok := s.Catalog.Course(e.CourseSlug)
		if !ok || cur.Onboarding == nil {
			continue
		}
		for _, q := range cur.LocalQuestions(lang) {
			o := cur.Option(e.Profile, q.ID)
			if o == nil {
				continue
			}
			answer := cur.LocalOption(o, lang).Label
			if t := e.Profile[curriculum.FreeTextKey(q.ID)]; o.FreeText && t != "" {
				answer = t
			}
			v.Profile = append(v.Profile, ProfileFact{Course: cur.Slug, CourseTitle: cur.Text(lang).Title, Question: q.Title, Answer: answer})
		}
	}
	items, err := storage.ListMemory(ctx, s.Store.Pool, userID, 100)
	if err != nil {
		return v, err
	}
	for _, m := range items {
		iv := MemoryItemView{MemoryItem: m}
		if cur, ok := s.Catalog.Course(m.CourseSlug); ok {
			if n := cur.LocalNode(m.SourceNode, lang); n != nil {
				iv.SourceTitle = n.Title
			}
		}
		v.Items = append(v.Items, iv)
	}
	return v, nil
}

func (s *Service) SetMemoryEnabled(ctx context.Context, userID string, on bool) (MemoryView, error) {
	if _, err := storage.SetMemoryEnabled(ctx, s.Store.Pool, userID, on); err != nil {
		return MemoryView{}, err
	}
	return s.Memory(ctx, userID)
}

func (s *Service) UpdateMemory(ctx context.Context, userID, id, text string) (MemoryView, error) {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" || utf8.RuneCountInString(text) > 200 {
		return MemoryView{}, errf(400, "invalid_memory", false, "A note needs 1–200 characters.")
	}
	if _, err := storage.UpdateMemory(ctx, s.Store.Pool, id, userID, text); errors.Is(err, storage.ErrNotFound) {
		return MemoryView{}, errf(404, "memory_not_found", false, "Note not found.")
	} else if err != nil {
		return MemoryView{}, err
	}
	return s.Memory(ctx, userID)
}

func (s *Service) DeleteMemory(ctx context.Context, userID, id string) (MemoryView, error) {
	ok, err := storage.DeleteMemory(ctx, s.Store.Pool, id, userID)
	if err != nil {
		return MemoryView{}, err
	}
	if !ok {
		return MemoryView{}, errf(404, "memory_not_found", false, "Note not found.")
	}
	return s.Memory(ctx, userID)
}

func (s *Service) DeleteAllMemory(ctx context.Context, userID string) (MemoryView, error) {
	if err := storage.DeleteAllMemory(ctx, s.Store.Pool, userID); err != nil {
		return MemoryView{}, err
	}
	return s.Memory(ctx, userID)
}

// ---- notes ("Catatanku") ----

type NoteInput struct {
	Course   string         `json:"course"`
	NodeID   string         `json:"nodeId"`
	Artifact tutor.Artifact `json:"artifact"`
}

type NoteView struct {
	storage.Note
	NodeTitle   string `json:"nodeTitle"`
	CourseTitle string `json:"courseTitle"`
}

const maxNoteBytes = 20 << 10

func (s *Service) SaveNote(ctx context.Context, userID string, in NoteInput) (NoteView, error) {
	cur, err := s.course(in.Course)
	if err != nil {
		return NoteView{}, err
	}
	if _, ok := cur.Node(in.NodeID); !ok {
		return NoteView{}, errf(404, "node_not_found", false, "No lesson %q.", in.NodeID)
	}
	a := in.Artifact
	a.Title = strings.TrimSpace(a.Title)
	if a.Title == "" || utf8.RuneCountInString(a.Title) > 200 || len(a.Rows) > 30 {
		return NoteView{}, errf(400, "invalid_note", false, "A note needs a title and a reasonable size.")
	}
	raw, err := json.Marshal(a)
	if err != nil || len(raw) > maxNoteBytes {
		return NoteView{}, errf(400, "invalid_note", false, "That note is too large.")
	}
	n, err := storage.SaveNote(ctx, s.Store.Pool, userID, cur.Slug, in.NodeID, a.Title, raw)
	if err != nil {
		return NoteView{}, err
	}
	return s.noteView(ctx, n), nil
}

func (s *Service) noteView(ctx context.Context, n storage.Note) NoteView {
	v := NoteView{Note: n}
	lang := langFrom(ctx)
	if cur, ok := s.Catalog.Course(n.CourseSlug); ok {
		v.CourseTitle = cur.Text(lang).Title
		if node := cur.LocalNode(n.NodeID, lang); node != nil {
			v.NodeTitle = node.Title
		}
	}
	return v
}

func (s *Service) Notes(ctx context.Context, userID string) ([]NoteView, error) {
	notes, err := storage.ListNotes(ctx, s.Store.Pool, userID)
	if err != nil {
		return nil, err
	}
	out := make([]NoteView, 0, len(notes))
	for _, n := range notes {
		out = append(out, s.noteView(ctx, n))
	}
	return out, nil
}

func (s *Service) DeleteNote(ctx context.Context, userID, id string) error {
	ok, err := storage.DeleteNote(ctx, s.Store.Pool, id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errf(404, "note_not_found", false, "Note not found.")
	}
	return nil
}

// ---- speech ----

const (
	maxSpeechRunes = 300
	MaxAudioBytes  = 2 << 20
)

var errNoAudio = errf(501, "speech_unavailable", false, "Speech isn't available on this server; the app will use the device's voice.")

// Speech returns MP3 audio for an English sentence, cached on disk so each
// sentence is only synthesised once.
func (s *Service) Speech(ctx context.Context, userID, text string) ([]byte, error) {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" || utf8.RuneCountInString(text) > maxSpeechRunes {
		return nil, errf(400, "invalid_text", false, "Text must be 1–%d characters.", maxSpeechRunes)
	}
	h := sha256.Sum256([]byte(s.AI.Name() + "\x00" + text))
	dir := s.SpeechCacheDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "fluentpath-tts")
	}
	path := filepath.Join(dir, hex.EncodeToString(h[:])+".mp3")
	if b, err := os.ReadFile(path); err == nil {
		return b, nil
	}
	if err := s.checkQuota(ctx, userID); err != nil {
		return nil, err
	}
	audio, err := s.AI.Speech(ctx, text)
	if errors.Is(err, ai.ErrNoAudio) {
		return nil, errNoAudio
	}
	if err != nil {
		slog.Warn("speech failed", "err", err)
		return nil, errf(502, "speech_failed", true, "Couldn't make the audio right now.")
	}
	// Count speech roughly as output tokens (about 4 characters per token).
	s.recordUsage(ctx, userID, "speech", ai.Usage{Output: utf8.RuneCountInString(text)/4 + 1})
	if err := os.MkdirAll(dir, 0o755); err == nil {
		if err := os.WriteFile(path, audio, 0o644); err != nil {
			slog.Warn("cache speech", "err", err)
		}
	}
	return audio, nil
}

// Transcribe turns a short recording of the learner into text.
func (s *Service) Transcribe(ctx context.Context, userID string, audio []byte, filename string) (string, error) {
	if len(audio) == 0 || len(audio) > MaxAudioBytes {
		return "", errf(400, "invalid_audio", false, "Recordings must be under 2 MB (about a minute).")
	}
	if err := s.checkQuota(ctx, userID); err != nil {
		return "", err
	}
	text, err := s.AI.Transcribe(ctx, audio, filename)
	if errors.Is(err, ai.ErrNoAudio) {
		return "", errNoAudio
	}
	if err != nil {
		slog.Warn("transcription failed", "err", err)
		return "", errf(502, "transcription_failed", true, "Couldn't understand the recording. Please try again.")
	}
	s.recordUsage(ctx, userID, "transcribe", ai.Usage{Input: len(audio)/1000 + 1})
	return text, nil
}
