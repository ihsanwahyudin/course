package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAskKeepsHistoryBeyondVisibleChats(t *testing.T) {
	var messages []ChatMessage
	o := fakeOpenAI(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []ChatMessage `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		messages = body.Input
		_, _ = io.WriteString(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{}"}]}]}`)
	})
	in, _ := lessonInput(t, "a0-alphabet-sounds", "id")
	var history []ChatMessage
	for i := 0; i < 7; i++ {
		history = append(history, ChatMessage{Role: "user", Content: fmt.Sprintf("Pertanyaan %d", i)}, ChatMessage{Role: "assistant", Content: fmt.Sprintf("Jawaban %d", i)})
	}
	const question = "Jelaskan lagi jawaban pertamamu."
	if _, err := o.Ask(context.Background(), AskInput{LessonInput: in, History: history, Question: question}); err != nil {
		t.Fatal(err)
	}
	if len(messages) != len(history)+3 {
		t.Fatalf("history was truncated: got %d messages, want %d", len(messages), len(history)+3)
	}
	for i, want := range history {
		if got := messages[i+1]; got != want {
			t.Fatalf("history message %d: got %+v, want %+v", i, got, want)
		}
	}
	if got := messages[len(messages)-1]; got.Role != "user" || got.Content != question {
		t.Fatalf("latest question: %+v", got)
	}
}

// Exercise the actual provider request: the language override must replace
// the conflicting UI policy in the system message, with the question intact.
func TestAskReplyLanguageOverridesUILanguage(t *testing.T) {
	for _, tc := range []struct {
		name, locale, question, fallback string
	}{
		{"Indonesian question in English UI", "en", "kenapa oo dibaca u?", "English"},
		{"English question in Indonesian UI", "id", "Why does book have two o letters?", "Indonesian (Bahasa Indonesia)"},
		{"Mixed question", "en", "kenapa book pakai double O?", "English"},
		{"Explicit language request", "en", "Explain double O in Indonesian", "English"},
		{"Ambiguous question", "id", "oo?", "Indonesian (Bahasa Indonesia)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body struct {
				Input []struct {
					Role, Content string
				} `json:"input"`
			}
			o := fakeOpenAI(t, func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				_, _ = io.WriteString(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"answer\":\"Penjelasan.\",\"scope\":\"in\",\"otherLesson\":\"\",\"remember\":\"\"}"}]}]}`)
			})
			in, _ := lessonInput(t, "a0-alphabet-sounds", tc.locale)
			_, err := o.Ask(context.Background(), AskInput{
				LessonInput: in, Question: tc.question,
				History: []ChatMessage{{Role: "user", Content: "What is double O?"}, {Role: "assistant", Content: "It means two O letters."}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(body.Input) != 5 || body.Input[0].Role != "system" {
				t.Fatalf("unexpected messages: %+v", body.Input)
			}
			system := body.Input[0].Content
			for _, want := range []string{
				"latest question, regardless of the app language or earlier replies",
				"write the entire explanation in Indonesian (Bahasa Indonesia)",
				"Follow an explicit request",
				"not quoted English words",
				"use " + tc.fallback + " as the default",
			} {
				if !strings.Contains(system, want) {
					t.Errorf("system message missing %q", want)
				}
			}
			if strings.Contains(system, in.LanguagePolicy) {
				t.Error("conflicting UI language policy remains in Ask AI system message")
			}
			if last := body.Input[len(body.Input)-1]; last.Role != "user" || last.Content != tc.question {
				t.Fatalf("latest question lost: %+v", last)
			}
		})
	}
}
