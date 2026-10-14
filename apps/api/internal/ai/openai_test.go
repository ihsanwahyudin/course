package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fakeOpenAI(t *testing.T, handler http.HandlerFunc) *OpenAI {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	o := NewOpenAI("test-key", "tutor-model", "", srv.URL)
	o.HTTP = &http.Client{Timeout: 5 * time.Second}
	return o
}

func TestOpenAILessonCallAndUsage(t *testing.T) {
	var body map[string]any
	o := fakeOpenAI(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = io.WriteString(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"hint\":\"Think about Andi.\"}"}]}],
			"usage":{"input_tokens":1500,"output_tokens":40,"input_tokens_details":{"cached_tokens":1280}}}`)
	})
	in, _ := lessonInput(t, "a0-pronouns", "en")
	in.CacheKey = "english:a0-pronouns:en"
	res, err := o.Hint(context.Background(), HintInput{LessonInput: in})
	if err != nil {
		t.Fatal(err)
	}
	if res.Usage != (Usage{Input: 1500, Cached: 1280, Output: 40}) || body["prompt_cache_key"] != "english:a0-pronouns:en" {
		t.Fatalf("usage %+v body %v", res.Usage, body["prompt_cache_key"])
	}
	roles := []string{}
	for _, m := range body["input"].([]any) {
		roles = append(roles, m.(map[string]any)["role"].(string))
	}
	if strings.Join(roles, ",") != "system,developer" {
		t.Fatalf("roles: %v", roles)
	}
}

func TestOpenAISpeechAndTranscription(t *testing.T) {
	var gotModel, gotFile string
	o := fakeOpenAI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/audio/speech":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			gotModel = b["model"].(string)
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("ID3fake"))
		case "/audio/transcriptions":
			_ = r.ParseMultipartForm(1 << 20)
			gotModel = r.FormValue("model")
			f, h, _ := r.FormFile("file")
			if f != nil {
				gotFile = h.Filename
			}
			_, _ = io.WriteString(w, `{"text":" They work in marketing. "}`)
		}
	})
	audio, err := o.Speech(context.Background(), "Hello")
	if err != nil || string(audio) != "ID3fake" || gotModel != "gpt-4o-mini-tts" {
		t.Fatalf("speech: %q %v %s", audio, err, gotModel)
	}
	text, err := o.Transcribe(context.Background(), []byte("webm"), "answer.webm")
	if err != nil || text != "They work in marketing." || gotModel != "gpt-4o-mini-transcribe" || gotFile != "answer.webm" {
		t.Fatalf("transcribe: %q %v %s %s", text, err, gotModel, gotFile)
	}
}
