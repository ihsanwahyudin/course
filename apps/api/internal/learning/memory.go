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
	"sync"
	"time"
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
	// The provider's speech endpoint is usually quick but sometimes takes
	// tens of seconds; a call is allowed this long to finish and be cached.
	speechTimeout = 90 * time.Second
	warmMaxTexts  = 60
	warmParallel  = 3
)

var errNoAudio = errf(501, "speech_unavailable", false, "Speech isn't available on this server; the app will use the device's voice.")

type speechCall struct {
	done  chan struct{}
	audio []byte
	err   error
}

func speechText(text string) string { return curriculum.SpeechKey(text) }

// say asks the provider for a sentence's audio.
func (s *Service) say(ctx context.Context, text string) ([]byte, error) {
	return s.AI.Speech(ctx, speakable(text))
}

// letterWords gives each letter a common word, so a lone letter is read as
// "A, as in apple." The voice garbles a bare letter (a clipped "uh" for A),
// and this is how letters are spelled out loud anyway.
var letterWords = map[byte]string{
	'A': "apple", 'B': "ball", 'C': "cat", 'D': "dog", 'E': "egg", 'F': "fish", 'G': "goat", 'H': "hat", 'I': "ink",
	'J': "jam", 'K': "kite", 'L': "lion", 'M': "moon", 'N': "nose", 'O': "orange", 'P': "pen", 'Q': "queen", 'R': "rain",
	'S': "sun", 'T': "tree", 'U': "umbrella", 'V': "van", 'W': "water", 'X': "box", 'Y': "yellow", 'Z': "zoo",
}

// speakable is what the voice is given for a sentence.
func speakable(text string) string {
	if len(text) == 1 {
		if w, ok := letterWords[text[0]]; ok {
			return text + ", as in " + w + "."
		}
	}
	return text
}

// bundledAudio returns a sentence's audio from a course folder (lesson
// audio is committed to the repo), if any.
func (s *Service) bundledAudio(text string) ([]byte, bool) {
	for _, cur := range s.Catalog.Courses {
		if b, err := os.ReadFile(cur.AudioPath(text)); err == nil {
			return b, true
		}
	}
	return nil, false
}

func (s *Service) speechPath(text string) string {
	voice := s.AI.Name()
	if v, ok := s.AI.(interface{ SpeechVoice() string }); ok {
		voice += "/" + v.SpeechVoice()
	}
	h := sha256.Sum256([]byte(voice + "\x00" + text))
	dir := s.SpeechCacheDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "fluentpath-tts")
	}
	return filepath.Join(dir, hex.EncodeToString(h[:])+".mp3")
}

// Speech returns MP3 audio for an English sentence, cached on disk so each
// sentence is only synthesised once.
func (s *Service) Speech(ctx context.Context, userID, text string) ([]byte, error) {
	return s.speech(ctx, userID, text, s.LiveSpeech)
}

// speech is Speech; live lets it ask the provider for audio it doesn't have
// (checkpoint listening questions are new every attempt, so they always may).
func (s *Service) speech(ctx context.Context, userID, text string, live bool) ([]byte, error) {
	text = speechText(text)
	if text == "" || utf8.RuneCountInString(text) > maxSpeechRunes {
		return nil, errf(400, "invalid_text", false, "Text must be 1–%d characters.", maxSpeechRunes)
	}
	if b, ok := s.bundledAudio(text); ok {
		return b, nil
	}
	path := s.speechPath(text)
	if b, err := os.ReadFile(path); err == nil {
		return b, nil
	}
	if !live {
		return nil, errNoAudio
	}
	if err := s.checkQuota(ctx, userID); err != nil {
		return nil, err
	}
	c := s.synthesize(ctx, userID, text, path)
	select {
	case <-c.done:
	case <-ctx.Done():
		// The audio still gets made and cached for next time.
		return nil, errf(502, "speech_failed", true, "Couldn't make the audio right now.")
	}
	if errors.Is(c.err, ai.ErrNoAudio) {
		return nil, errNoAudio
	}
	if c.err != nil {
		slog.Warn("speech failed", "err", c.err)
		return nil, errf(502, "speech_failed", true, "Couldn't make the audio right now.")
	}
	return c.audio, nil
}

// synthesize starts (or joins) the provider call for one sentence. Concurrent
// requests share one call, and the call finishes and is cached even if the
// requester goes away.
func (s *Service) synthesize(ctx context.Context, userID, text, path string) *speechCall {
	s.speechMu.Lock()
	defer s.speechMu.Unlock()
	if c, ok := s.speechInflight[path]; ok {
		return c
	}
	if s.speechInflight == nil {
		s.speechInflight = map[string]*speechCall{}
	}
	c := &speechCall{done: make(chan struct{})}
	s.speechInflight[path] = c
	go func() {
		defer func() {
			s.speechMu.Lock()
			delete(s.speechInflight, path)
			s.speechMu.Unlock()
			close(c.done)
		}()
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), speechTimeout)
		defer cancel()
		c.audio, c.err = s.say(cctx, text)
		if c.err != nil {
			return
		}
		// Count speech roughly as output tokens (about 4 characters per token).
		s.recordUsage(cctx, userID, "speech", ai.Usage{Output: utf8.RuneCountInString(text)/4 + 1})
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
			if err := os.WriteFile(path, c.audio, 0o644); err != nil {
				slog.Warn("cache speech", "err", err)
			}
		}
	}()
	return c
}

// warmSpeech makes a new plan's audio in the background.
func (s *Service) warmSpeech(userID string, p tutor.Plan) {
	if s.WarmSpeech && s.LiveSpeech {
		s.warmTexts(userID, tutor.SpeechTexts(p, warmMaxTexts), s.LiveSpeech)
	}
}

// WarmAllLessons makes, in the background, the audio of any lesson sentence
// that has no recording in the repo yet (normally none: run
// `go run ./cmd/server bundle-audio` after changing lessons). Run at startup.
func (s *Service) WarmAllLessons() {
	var missing []string
	total := 0
	for _, cur := range s.Catalog.Courses {
		for _, t := range tutor.LessonSpeechTexts(cur) {
			total++
			if _, ok := s.bundledAudio(t); ok {
				continue
			}
			if _, err := os.Stat(s.speechPath(t)); err != nil {
				missing = append(missing, t)
			}
		}
	}
	slog.Info("lesson audio", "sentences", total, "missing", len(missing))
	if len(missing) > 0 {
		slog.Warn("lesson sentences without recorded audio; run `go run ./cmd/server bundle-audio` and commit the files", "count", len(missing))
	}
	if !s.WarmSpeech || !s.LiveSpeech {
		return
	}
	// No learner: shared lesson audio doesn't count against anyone's budget.
	s.warmTexts("", missing, s.LiveSpeech)
}

// BundleLessonAudio writes the audio of every lesson sentence into its
// course's audio folder (for committing to the repo). Sentences that already
// have a file are skipped; the speech cache is used before the provider.
// It returns how many files were written and the sentences that failed.
// fresh re-records every sentence straight from the provider (after
// changing the voice or model), ignoring existing files and the cache.
func (s *Service) BundleLessonAudio(ctx context.Context, fresh bool) (written int, failed []string) {
	for _, cur := range s.Catalog.Courses {
		var todo []string
		for _, t := range tutor.LessonSpeechTexts(cur) {
			if _, err := os.Stat(cur.AudioPath(t)); fresh || err != nil {
				todo = append(todo, t)
			}
		}
		for round := 0; round < 3 && len(todo) > 0; round++ {
			var again []string
			var mu sync.Mutex
			var wg sync.WaitGroup
			queue := make(chan string)
			for range bundleWorkers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for t := range queue {
						err := s.bundleOne(ctx, cur, t, fresh)
						mu.Lock()
						if err != nil {
							again = append(again, t)
						} else if written++; written%100 == 0 {
							slog.Info("lesson audio", "written", written)
						}
						mu.Unlock()
					}
				}()
			}
			for _, t := range todo {
				queue <- t
			}
			close(queue)
			wg.Wait()
			todo = again
		}
		failed = append(failed, todo...)
		pruneAudio(cur)
	}
	return written, failed
}

// bundleWorkers is how many sentences are recorded at once.
const bundleWorkers = 12

func (s *Service) bundleOne(ctx context.Context, cur *curriculum.Curriculum, t string, fresh bool) error {
	var b []byte
	var err error
	if fresh {
		cctx, cancel := context.WithTimeout(ctx, speechTimeout)
		b, err = s.say(cctx, t)
		cancel()
	} else {
		b, err = s.Speech(ctx, "", t)
	}
	if err == nil {
		err = os.MkdirAll(filepath.Dir(cur.AudioPath(t)), 0o755)
	}
	if err == nil {
		err = os.WriteFile(cur.AudioPath(t), b, 0o644)
	}
	return err
}

// pruneAudio removes recordings no lesson reads any more (after edits).
func pruneAudio(cur *curriculum.Curriculum) {
	keep := map[string]bool{}
	for _, t := range tutor.LessonSpeechTexts(cur) {
		keep[filepath.Base(cur.AudioPath(t))] = true
	}
	dir := filepath.Join(cur.Dir, "audio")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp3") && !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				slog.Info("removed unused lesson audio", "file", e.Name())
			}
		}
	}
}

func (s *Service) warmTexts(userID string, texts []string, live bool) {
	if len(texts) == 0 {
		return
	}
	s.warm.Add(1)
	go func() {
		defer s.warm.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		// The provider sometimes times out; whatever failed gets two more rounds.
		todo := texts
		for round := 0; round < 3 && len(todo) > 0 && ctx.Err() == nil; round++ {
			todo = s.speakAll(ctx, userID, todo, live)
		}
		if userID == "" {
			slog.Info("lesson audio ready", "made", len(texts)-len(todo), "failed", len(todo))
		}
	}()
}

// speakAll makes audio for texts a few at a time and returns the ones that failed.
func (s *Service) speakAll(ctx context.Context, userID string, texts []string, live bool) []string {
	sem := make(chan struct{}, warmParallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string
	for _, t := range texts {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := s.speech(ctx, userID, t, live); err != nil {
				mu.Lock()
				failed = append(failed, t)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return failed
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
