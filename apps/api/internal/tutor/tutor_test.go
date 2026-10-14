package tutor

import (
	"errors"
	"strings"
	"testing"
)

func TestParseReplyKeepsValidParts(t *testing.T) {
	raw := `{"message":" Here you go ","quiz":[
		{"prompt":"She ___ a nurse.","options":["am","is","are"],"answer":1,"explain":"she → is"},
		{"prompt":"bad answer","options":["a","b"],"answer":2,"explain":""},
		{"prompt":"one option","options":["a"],"answer":0,"explain":""}],
	"flashcards":[{"front":"I","back":"am"},{"front":"","back":"x"},{"front":"` + strings.Repeat("x", 200) + `","back":"y"}],
	"search":{"kind":"video","query":"verb to be"}}`
	r, err := ParseReply([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if r.Message != "Here you go" || len(r.Quiz) != 1 || r.Quiz[0].Answer != 1 || len(r.Flashcards) != 1 || r.Search.Kind != "video" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseReplyLimitsAndRejects(t *testing.T) {
	q := `{"prompt":"p","options":["a","b"],"answer":0,"explain":""}`
	raw := `{"message":"m","quiz":[` + strings.Repeat(q+",", 7) + q + `],"flashcards":[],"search":{"kind":"podcast","query":"x"}}`
	r, err := ParseReply([]byte(raw))
	if err != nil || len(r.Quiz) != MaxQuiz || r.Search.Kind != "" {
		t.Fatalf("limits: %+v %v", r, err)
	}
	for _, bad := range []string{`{"message":"  "}`, `not json`, `{"message":"` + strings.Repeat("x", 2001) + `"}`} {
		if _, err := ParseReply([]byte(bad)); !errors.Is(err, ErrMalformed) {
			t.Errorf("%q should be malformed, got %v", bad, err)
		}
	}
}

func TestIntents(t *testing.T) {
	if !IntentQuiz.Valid() || Intent("dance").Valid() || IntentVideo.SearchKind() != KindVideo || IntentAsk.SearchKind() != "" {
		t.Fatal("intent helpers wrong")
	}
	if IntentQuiz.MaxOutputTokens() <= IntentAsk.MaxOutputTokens() {
		t.Fatal("quizzes need more output room than plain answers")
	}
}
