package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/auth"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/learning"
	"fluentpath/api/internal/tutor"
)

// Explicit opt-in only. Requests may go only to Responses; audio is blocked
// independently of service settings. This prevents a live check from spending
// any TTS credits, including if a future regression enables warming.
type liveBudgetTransport struct {
	mu                          sync.Mutex
	calls, tokens, audioBlocked int
}

func (b *liveBudgetTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !strings.HasSuffix(r.URL.Path, "/responses") {
		b.audioBlocked++
		return nil, errors.New("live checkpoint check blocks non-Responses endpoints")
	}
	if b.calls >= 6 || b.tokens >= 20000 {
		return nil, errors.New("live checkpoint test budget exhausted")
	}
	b.calls++
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	var usage struct {
		Usage struct {
			Input   int `json:"input_tokens"`
			Output  int `json:"output_tokens"`
			Details struct {
				Cached int `json:"cached_tokens"`
			} `json:"input_tokens_details"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &usage) == nil {
		b.tokens += usage.Usage.Input - usage.Usage.Details.Cached + usage.Usage.Output
	}
	response.Body = io.NopCloser(bytes.NewReader(raw))
	return response, nil
}

func TestLiveCheckpointHTTP(t *testing.T) {
	if os.Getenv("LIVE_OPENAI_CHECKPOINT_TEST") != "1" {
		t.Skip("explicit live OpenAI opt-in required")
	}
	// Read only the needed server configuration; never print the file or keys.
	config := map[string]string{}
	raw, err := os.ReadFile("../../.env")
	if err != nil {
		t.Fatal("server .env unavailable")
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok {
			config[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	if config["OPENAI_API_KEY"] == "" || config["OPENAI_MODEL"] == "" {
		t.Fatal("server OpenAI configuration missing")
	}
	_, store := dbRouter(t)
	cat, err := curriculum.LoadCatalog("../../../../packages/courses")
	if err != nil {
		t.Fatal(err)
	}
	provider := ai.NewOpenAI(config["OPENAI_API_KEY"], config["OPENAI_MODEL"], config["OPENAI_EVAL_MODEL"], config["OPENAI_BASE_URL"])
	provider.MaxRetries = 0
	budget := &liveBudgetTransport{}
	provider.HTTP.Transport = budget
	svc := &learning.Service{Store: store, Catalog: cat, AI: provider, TimeZone: "Asia/Jakarta", DailyTokens: 20000, LiveSpeech: false, WarmSpeech: false, PrefetchCheckpoints: false}
	router := (&Server{Svc: svc, Auth: &auth.Service{Store: store}, Catalog: cat, Ping: store.Pool.Ping, RateLimit: 10000, AIRateLimit: 10000}).Router()
	token := ""
	request := func(method, path string, payload any) map[string]any {
		t.Helper()
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Language", "id")
		req.Header.Set(nativeClientHeader, "native")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if rec.Code != 200 {
			t.Fatalf("live HTTP %s %s: status %d", method, path, rec.Code)
		}
		return out
	}
	account := request("POST", "/api/v1/auth/register", map[string]string{"email": "live-check@example.com", "password": "local-live-check-123", "displayName": "Local Check"})
	token = account["token"].(string)
	type result struct {
		Node   string            `json:"node"`
		Source string            `json:"source"`
		Items  []tutor.CheckItem `json:"items"`
		Review any               `json:"review"`
	}
	results := []result{}
	defer func() {
		dir := "../../../../review"
		_ = os.MkdirAll(dir, 0755)
		data, _ := json.MarshalIndent(map[string]any{"model": config["OPENAI_MODEL"], "generationOnly": os.Getenv("LIVE_CHECKPOINT_GENERATION_ONLY") == "1", "calls": budget.calls, "uncachedTokensAndOutput": budget.tokens, "blockedNonResponsesRequests": budget.audioBlocked, "results": results}, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "checkpoint-live-results.json"), append(data, '\n'), 0644); err != nil {
			t.Error(err)
		}
		t.Logf("Live calls=%d, uncached input + output=%d, non-Responses requests=%d", budget.calls, budget.tokens, budget.audioBlocked)
	}()
	nodes := []string{"a0-alphabet-sounds", "a0-greetings"}
	if os.Getenv("LIVE_CHECKPOINT_GENERATION_ONLY") == "1" {
		nodes = nodes[:1]
	}
	for _, node := range nodes {
		sess := request("POST", "/api/v1/sessions", map[string]string{"course": "english", "nodeId": node})
		sid := sess["id"].(string)
		for {
			lesson := sess["lesson"].(map[string]any)
			if lesson["checkpoint"].(bool) {
				break
			}
			index := int(lesson["stepIndex"].(float64))
			step := lesson["steps"].([]any)[index].(map[string]any)
			if q, ok := step["question"].(map[string]any); ok && !q["answered"].(bool) && !q["revealed"].(bool) {
				answer := map[string]any{"choice": 0}
				if q["type"] != "mcq" {
					answer = map[string]any{"tokens": q["tokens"]}
				}
				sess = request("POST", fmt.Sprintf("/api/v1/sessions/%s/steps/%d/answer", sid, index), answer)
			} else {
				sess = request("POST", fmt.Sprintf("/api/v1/sessions/%s/steps/%d/next", sid, index), map[string]any{})
			}
		}
		cp := request("POST", "/api/v1/sessions/"+sid+"/checkpoint", map[string]any{})
		id := cp["id"].(string)
		var itemJSON []byte
		var source string
		if err := store.Pool.QueryRow(context.Background(), "SELECT items, source FROM checkpoints WHERE id=$1", id).Scan(&itemJSON, &source); err != nil {
			t.Fatal(err)
		}
		var items []tutor.CheckItem
		if err := json.Unmarshal(itemJSON, &items); err != nil {
			t.Fatal(err)
		}
		results = append(results, result{Node: node, Source: source, Items: items})
		if source != "openai" {
			t.Fatalf("%s used %s fallback instead of verified AI output; inspect report", node, source)
		}
		if err := tutor.ValidateLevelQuestions(items, "A0"); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("LIVE_CHECKPOINT_GENERATION_ONLY") == "1" {
			continue
		}
		if budget.calls >= 6 || budget.tokens >= 20000 {
			t.Fatal("budget leaves no room for live grading; generation verified but grading is incomplete")
		}
		answers := []map[string]string{}
		for _, q := range items {
			answer := "wrong answer"
			switch q.Type {
			case tutor.QuestionMCQ:
				answer = fmt.Sprint((q.Answer + 1) % len(q.Options))
			case tutor.QuestionMulti:
				for i := range q.Options {
					found := false
					for _, a := range q.Answers {
						if a == i {
							found = true
						}
					}
					if !found {
						answer = fmt.Sprint(i)
						break
					}
				}
			case tutor.QuestionMatch:
				picks := append(append([]int{}, q.Pairing[1:]...), q.Pairing[0])
				parts := []string{}
				for _, p := range picks {
					parts = append(parts, fmt.Sprint(p))
				}
				answer = strings.Join(parts, ",")
			case tutor.QuestionLetters:
				answer = q.Tokens[0]
			case tutor.QuestionWordOrder:
				words := strings.Fields(q.Solution)
				for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
					words[i], words[j] = words[j], words[i]
				}
				answer = strings.Join(words, " ")
			}
			answers = append(answers, map[string]string{"itemId": q.ID, "answer": answer})
		}
		graded := request("POST", "/api/v1/sessions/"+sid+"/assessment", map[string]any{"checkpointId": id, "answers": answers})
		results[len(results)-1].Review = graded["attempt"].(map[string]any)["review"]
		if graded["passed"].(bool) {
			t.Fatal("intentionally wrong checkpoint must not master the lesson")
		}
	}
	if budget.audioBlocked != 0 {
		t.Fatal("unexpected audio attempt was blocked")
	}
}
