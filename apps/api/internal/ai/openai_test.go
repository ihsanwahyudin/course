package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fluentpath/api/internal/tutor"
)

func fakeOpenAI(t *testing.T, reply string, seen *map[string]any) *OpenAI {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if seen != nil {
			_ = json.Unmarshal(body, seen)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	o := NewOpenAI("test-key", "tutor-model", "", srv.URL)
	o.HTTP = &http.Client{Timeout: 5 * time.Second}
	return o
}

func TestOpenAITutorRequestAndUsage(t *testing.T) {
	var body map[string]any
	o := fakeOpenAI(t, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"message\":\"hi\",\"quiz\":[],\"flashcards\":[],\"search\":{\"kind\":\"\",\"query\":\"\"}}"}]}],
		"usage":{"input_tokens":1500,"output_tokens":40,"input_tokens_details":{"cached_tokens":1280}}}`, &body)
	in, _ := lessonInput(t, "a1-to-be")
	in.Intent, in.CacheKey = tutor.IntentQuiz, "english:a1-to-be:id"
	in.Messages = []ChatMessage{{Role: "user", Content: "earlier"}, {Role: "assistant", Content: "answer"}, {Role: "user", Content: "quiz me"}}
	res, err := o.TutorReply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Usage != (Usage{Input: 1500, Cached: 1280, Output: 40}) {
		t.Fatalf("usage not parsed: %+v", res.Usage)
	}
	if body["prompt_cache_key"] != "english:a1-to-be:id" || body["max_output_tokens"].(float64) != float64(tutor.IntentQuiz.MaxOutputTokens()) {
		t.Fatalf("request missing cache key or token cap: %v", body)
	}
	input := body["input"].([]any)
	roles := []string{}
	for _, m := range input {
		roles = append(roles, m.(map[string]any)["role"].(string))
	}
	want := []string{"system", "user", "assistant", "developer", "user"}
	if len(roles) != len(want) {
		t.Fatalf("roles %v, want %v", roles, want)
	}
	for i := range want {
		if roles[i] != want[i] {
			t.Fatalf("roles %v, want %v", roles, want)
		}
	}
}

func TestOpenAIWebSearchKeepsOnlyCitedURLs(t *testing.T) {
	var body map[string]any
	o := fakeOpenAI(t, `{"status":"completed","output":[{"type":"web_search_call"},{"type":"message","content":[{"type":"output_text",
		"text":"See https://made-up.example/video and the cited ones.",
		"annotations":[
			{"type":"url_citation","url":"https://www.youtube.com/watch?v=wlbQLBhLk_4&utm_source=openai","title":"Alphabet song"},
			{"type":"url_citation","url":"https://www.youtube.com/watch?v=wlbQLBhLk_4","title":"dup"},
			{"type":"url_citation","url":"https://example.com/not-a-video","title":"page"},
			{"type":"url_citation","url":"http://youtube.com/watch?v=b3L4pLSUwig","title":"insecure"}]}]}],
		"usage":{"input_tokens":300,"output_tokens":80}}`, &body)
	links, usage, err := o.WebSearch(context.Background(), SearchInput{Kind: tutor.KindVideo, Query: "alphabet song", LessonTitle: "Alphabet"})
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].URL != "https://www.youtube.com/watch?v=wlbQLBhLk_4" || links[0].Source != "youtube.com" || usage.Input != 300 {
		t.Fatalf("links %+v usage %+v", links, usage)
	}
	tools := body["tools"].([]any)
	if tools[0].(map[string]any)["type"] != "web_search" || tools[0].(map[string]any)["search_context_size"] != "low" {
		t.Fatalf("web_search tool not requested: %v", body["tools"])
	}
	o.DisableSearch = true
	if _, _, err := o.WebSearch(context.Background(), SearchInput{Kind: tutor.KindArticle}); err != ErrNoSearch {
		t.Fatal("WEB_SEARCH=off must disable search")
	}
}
