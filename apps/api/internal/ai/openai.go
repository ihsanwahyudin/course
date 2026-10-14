package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func (o *OpenAI) TutorReply(ctx context.Context, in TutorInput) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, o.TutorBudget)
	defer cancel()
	input := []map[string]string{{"role": "system", "content": tutorSystemPrompt(in)}}
	// Keep the request small: the system prompt carries the lesson context.
	msgs := in.Messages
	if len(msgs) > 24 {
		msgs = msgs[len(msgs)-24:]
	}
	for _, m := range msgs {
		input = append(input, map[string]string{"role": m.Role, "content": m.Content})
	}
	return o.respond(ctx, o.TutorModel, input, "tutor_reply", tutorSchema(), 800)
}

func (o *OpenAI) Evaluate(ctx context.Context, in EvalInput) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, o.EvalBudget)
	defer cancel()
	input := []map[string]string{
		{"role": "system", "content": evalSystemPrompt(in)},
		{"role": "user", "content": evalUserPrompt(in)},
	}
	return o.respond(ctx, o.EvalModel, input, "assessment_result", evalSchema(in), 1200)
}

type responsesReply struct {
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Refusal string `json:"refusal"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *OpenAI) respond(ctx context.Context, model string, input any, name string, schema map[string]any, maxTokens int) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"model": model,
		"input": input,
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   name,
				"schema": schema,
				"strict": true,
			},
		},
		"max_output_tokens": maxTokens,
		"store":             false,
	})
	if err != nil {
		return nil, err
	}

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

func (o *OpenAI) once(ctx context.Context, body []byte) (json.RawMessage, bool, error) {
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
	for _, item := range r.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			switch c.Type {
			case "output_text":
				return json.RawMessage(c.Text), false, nil
			case "refusal":
				return nil, false, errors.New("model refused the request")
			}
		}
	}
	return nil, false, errors.New("openai response had no output text")
}
