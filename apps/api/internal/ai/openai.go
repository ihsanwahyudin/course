package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
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
	// PlanBudget bounds writing a whole lesson plan (a long reply).
	PlanBudget time.Duration
	// Speech models (configurable; see .env.example).
	TTSModel        string
	TTSVoice        string
	TranscribeModel string
}

func NewOpenAI(apiKey, tutorModel, evalModel, baseURL string) *OpenAI {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if evalModel == "" {
		evalModel = tutorModel
	}
	return &OpenAI{
		APIKey:          apiKey,
		TutorModel:      tutorModel,
		EvalModel:       evalModel,
		BaseURL:         strings.TrimRight(baseURL, "/"),
		HTTP:            &http.Client{Timeout: 150 * time.Second},
		MaxRetries:      2,
		TutorBudget:     45 * time.Second,
		EvalBudget:      60 * time.Second,
		PlanBudget:      150 * time.Second,
		TTSModel:        "gpt-4o-mini-tts",
		TTSVoice:        "alloy",
		TranscribeModel: "gpt-4o-mini-transcribe",
	}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) lessonCall(ctx context.Context, in LessonInput, task, user, name string, schema map[string]any, maxTokens int, history []ChatMessage) (Result, error) {
	budget := o.TutorBudget
	if name == "lesson_plan" {
		budget = o.PlanBudget
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	// The lesson prefix comes first and never changes between calls, so the
	// provider can reuse its prompt cache; the task goes after it.
	input := []map[string]string{{"role": "system", "content": lessonPrefix(in)}}
	for _, m := range history {
		input = append(input, map[string]string{"role": m.Role, "content": m.Content})
	}
	input = append(input, map[string]string{"role": "developer", "content": task})
	if user != "" {
		input = append(input, map[string]string{"role": "user", "content": user})
	}
	return o.respond(ctx, request{Model: o.TutorModel, Input: input, Name: name, Schema: schema, MaxTokens: maxTokens, CacheKey: in.CacheKey})
}

func (o *OpenAI) LessonPlan(ctx context.Context, in LessonInput) (Result, error) {
	return o.lessonCall(ctx, in, planTask(in), "", "lesson_plan", planSchema(), 6000, nil)
}

func (o *OpenAI) Hint(ctx context.Context, in HintInput) (Result, error) {
	return o.lessonCall(ctx, in.LessonInput, hintTask(in), "", "hint", hintSchema(), 300, nil)
}

func (o *OpenAI) SimilarQuestion(ctx context.Context, in SimilarInput) (Result, error) {
	return o.lessonCall(ctx, in.LessonInput, similarTask(in), "", "question", questionSchema(), 600, nil)
}

func (o *OpenAI) Ask(ctx context.Context, in AskInput) (Result, error) {
	h := in.History
	if len(h) > 8 {
		h = h[len(h)-8:]
	}
	return o.lessonCall(ctx, in.LessonInput, askTask(in), in.Question, "answer", answerSchema(), 700, h)
}

func (o *OpenAI) Summarize(ctx context.Context, in SummaryInput) (Result, error) {
	return o.lessonCall(ctx, in.LessonInput, summaryTask(in), "", "summary", summarySchema(), 700, nil)
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

// Speech calls the text-to-speech endpoint and returns MP3 audio.
func (o *OpenAI) Speech(ctx context.Context, text string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req := map[string]any{"model": o.TTSModel, "voice": o.TTSVoice, "input": text, "response_format": "mp3"}
	// tts-1 and tts-1-hd (billed per character, cheaper for short words)
	// don't take speaking instructions.
	if !strings.HasPrefix(o.TTSModel, "tts-1") {
		req["instructions"] = "Speak clearly and a little slowly, like an English teacher for beginners."
	}
	body, _ := json.Marshal(req)
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Authorization", "Bearer "+o.APIKey)
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := o.HTTP.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	audio, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: openai speech status %d: %s", ErrUnavailable, resp.StatusCode, apiError(audio))
	}
	return audio, nil
}

// Transcribe sends a short recording to the transcription endpoint.
func (o *OpenAI) Transcribe(ctx context.Context, audio []byte, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", o.TranscribeModel)
	_ = w.WriteField("language", "en")
	_ = w.WriteField("response_format", "json")
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(audio); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: openai transcription status %d: %s", ErrUnavailable, resp.StatusCode, apiError(raw))
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("%w: decode transcription: %v", ErrUnavailable, err)
	}
	return strings.TrimSpace(out.Text), nil
}

// apiError extracts OpenAI's error message (never the request contents).
func apiError(raw []byte) string {
	var e struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &e) == nil && e.Error != nil {
		return e.Error.Message
	}
	return ""
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
