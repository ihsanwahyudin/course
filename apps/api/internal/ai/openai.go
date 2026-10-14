package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fluentpath/api/internal/curriculum"
)

// OpenAI calls the Responses API with strict JSON-schema structured output.
type OpenAI struct {
	APIKey      string
	TutorModel  string
	EvalModel   string
	BaseURL     string // default https://api.openai.com/v1
	HTTP        *http.Client
	MaxRetries  int
	TutorBudget time.Duration
	EvalBudget  time.Duration
	// DisableSearch turns off the web_search tool (WEB_SEARCH=off).
	DisableSearch bool
}

func NewOpenAI(apiKey, tutorModel, evalModel, baseURL string) *OpenAI {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if evalModel == "" {
		evalModel = tutorModel
	}
	return &OpenAI{
		APIKey:      apiKey,
		TutorModel:  tutorModel,
		EvalModel:   evalModel,
		BaseURL:     strings.TrimRight(baseURL, "/"),
		HTTP:        &http.Client{Timeout: 60 * time.Second},
		MaxRetries:  2,
		TutorBudget: 45 * time.Second,
		EvalBudget:  60 * time.Second,
	}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) TutorReply(ctx context.Context, in TutorInput) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, o.TutorBudget)
	defer cancel()
	// Static prompt first so the provider can cache it; the per-turn
	// instruction goes right before the learner's newest message.
	input := []map[string]string{{"role": "system", "content": tutorStaticPrompt(in)}}
	msgs := in.Messages
	var last *ChatMessage
	if n := len(msgs); n > 0 && msgs[n-1].Role == "user" {
		last, msgs = &msgs[n-1], msgs[:n-1]
	}
	for _, m := range msgs {
		input = append(input, map[string]string{"role": m.Role, "content": m.Content})
	}
	input = append(input, map[string]string{"role": "developer", "content": tutorTurnPrompt(in)})
	if last != nil {
		input = append(input, map[string]string{"role": "user", "content": last.Content})
	}
	return o.respond(ctx, request{Model: o.TutorModel, Input: input, Name: "tutor_reply", Schema: tutorSchema(),
		MaxTokens: in.Intent.MaxOutputTokens(), CacheKey: in.CacheKey})
}

func (o *OpenAI) Evaluate(ctx context.Context, in EvalInput) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, o.EvalBudget)
	defer cancel()
	input := []map[string]string{
		{"role": "system", "content": evalSystemPrompt(in)},
		{"role": "user", "content": evalUserPrompt(in)},
	}
	return o.respond(ctx, request{Model: o.EvalModel, Input: input, Name: "assessment_result", Schema: evalSchema(in), MaxTokens: 1200})
}

// WebSearch uses the Responses API web_search tool and keeps only the URLs
// the tool actually cited, so a link can never be made up by the model.
func (o *OpenAI) WebSearch(ctx context.Context, in SearchInput) ([]Link, Usage, error) {
	if o.DisableSearch {
		return nil, Usage{}, ErrNoSearch
	}
	ctx, cancel := context.WithTimeout(ctx, o.TutorBudget)
	defer cancel()
	what := "real-world pages (news, guides, everyday examples) that show this English in use"
	if in.Kind == "video" {
		what = "beginner-friendly YouTube videos (youtube.com) that teach or show this"
	}
	prompt := fmt.Sprintf("Find up to 4 %s for an English learner studying %q. Search for: %s. "+
		"Prefer well-known educational sources. Cite every result you mention.", what, in.LessonTitle, in.Query)
	tool := map[string]any{"type": "web_search", "search_context_size": "low"}
	if in.Kind == "video" {
		tool["filters"] = map[string]any{"allowed_domains": []string{"youtube.com"}}
	}
	body := map[string]any{
		"model":             o.TutorModel,
		"input":             prompt,
		"tools":             []map[string]any{tool},
		"max_output_tokens": 400,
		"store":             false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, Usage{}, err
	}
	r, err := o.call(ctx, raw)
	if err != nil {
		return nil, Usage{}, err
	}
	var links []Link
	seen := map[string]bool{}
	for _, item := range r.Output {
		for _, c := range item.Content {
			for _, a := range c.Annotations {
				if a.Type != "url_citation" {
					continue
				}
				u := cleanURL(a.URL)
				if u == "" || seen[u] || len(links) >= 4 {
					continue
				}
				if in.Kind == "video" {
					if _, ok := curriculum.YouTubeID(u); !ok {
						continue
					}
				}
				seen[u] = true
				links = append(links, Link{Kind: in.Kind, Title: strings.TrimSpace(a.Title), URL: u, Source: hostOf(u)})
			}
		}
	}
	return links, r.usage(), nil
}

// cleanURL keeps https URLs and drops tracking parameters.
func cleanURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return ""
	}
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(k, "utm_") {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String()
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

type responsesReply struct {
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Refusal     string `json:"refusal"`
			Annotations []struct {
				Type  string `json:"type"`
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"annotations"`
		} `json:"content"`
	} `json:"output"`
	Usage *struct {
		InputTokens        int `json:"input_tokens"`
		OutputTokens       int `json:"output_tokens"`
		InputTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (r *responsesReply) usage() Usage {
	if r.Usage == nil {
		return Usage{}
	}
	u := Usage{Input: r.Usage.InputTokens, Output: r.Usage.OutputTokens}
	if r.Usage.InputTokensDetails != nil {
		u.Cached = r.Usage.InputTokensDetails.CachedTokens
	}
	return u
}

type request struct {
	Model     string
	Input     any
	Name      string
	Schema    map[string]any
	MaxTokens int
	CacheKey  string
}

func (o *OpenAI) respond(ctx context.Context, req request) (Result, error) {
	body := map[string]any{
		"model": req.Model,
		"input": req.Input,
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   req.Name,
				"schema": req.Schema,
				"strict": true,
			},
		},
		"max_output_tokens": req.MaxTokens,
		"store":             false,
	}
	if req.CacheKey != "" {
		body["prompt_cache_key"] = req.CacheKey
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}
	r, err := o.call(ctx, raw)
	if err != nil {
		return Result{}, err
	}
	for _, item := range r.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			switch c.Type {
			case "output_text":
				return Result{Raw: json.RawMessage(c.Text), Usage: r.usage()}, nil
			case "refusal":
				return Result{}, fmt.Errorf("%w: model refused the request", ErrUnavailable)
			}
		}
	}
	return Result{}, fmt.Errorf("%w: openai response had no output text", ErrUnavailable)
}

// call posts to /responses with retries on transient failures.
func (o *OpenAI) call(ctx context.Context, body []byte) (*responsesReply, error) {
	var lastErr error
	for attempt := 0; attempt <= o.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * 600 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
			case <-time.After(backoff):
			}
		}
		out, retry, err := o.once(ctx, body)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !retry {
			break
		}
	}
	return nil, fmt.Errorf("%w: %v", ErrUnavailable, lastErr)
}

func (o *OpenAI) once(ctx context.Context, body []byte) (*responsesReply, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return nil, ctx.Err() == nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, true, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("openai status %d", resp.StatusCode)
	}
	var r responsesReply
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, false, fmt.Errorf("decode openai response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := ""
		if r.Error != nil {
			msg = r.Error.Message
		}
		// Never echo request contents or credentials; the message is OpenAI's.
		return nil, false, fmt.Errorf("openai status %d: %s", resp.StatusCode, msg)
	}
	if r.Status == "incomplete" {
		reason := ""
		if r.IncompleteDetails != nil {
			reason = r.IncompleteDetails.Reason
		}
		return nil, false, fmt.Errorf("openai response incomplete: %s", reason)
	}
	if u := r.usage(); u.Input > 0 {
		slog.Info("openai usage", "input", u.Input, "cached", u.Cached, "output", u.Output)
	}
	return &r, false, nil
}
