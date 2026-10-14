package learning

import (
	"context"
	"fmt"
)

type langKey struct{}

// WithLang sets the UI language ("en" or "id") for a request.
func WithLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, langKey{}, lang)
}

func langFrom(ctx context.Context) string {
	if l, ok := ctx.Value(langKey{}).(string); ok && l != "" {
		return l
	}
	return "en"
}

// Server-written learner-facing text. Course content has its own i18n in
// the course files; errors are translated by clients from their codes.
var messages = map[string]map[string]string{
	"en": {
		"coming_soon":    "This lesson is coming soon.",
		"master_first":   "Master first: %s.",
		"mastered":       "You scored %d/100 and mastered %q. %s",
		"review":         "You scored %d/100 — you need %d to master this lesson. %s Review: %s. Re-read those parts or ask me about them, then try the final quiz again.",
		"exact_evidence": "Every answer matches a reference answer.",
		"exact_feedback": "Excellent — every answer is correct.",
		// Text stored for quick-action requests (shown as the learner's message).
		"intent_simpler":    "Explain this lesson more simply.",
		"intent_quiz":       "Give me a practice quiz.",
		"intent_flashcards": "Make flashcards for this lesson.",
		"intent_examples":   "Show me real-world examples.",
		"intent_video":      "Find a video about this.",
		"intent_image":      "Show me pictures for this.",
	},
	"id": {
		"coming_soon":       "Pelajaran ini segera hadir.",
		"master_first":      "Kuasai dulu: %s.",
		"mastered":          "Nilaimu %d/100 dan kamu menguasai %q. %s",
		"review":            "Nilaimu %d/100 — kamu butuh %d untuk menguasai pelajaran ini. %s Ulas lagi: %s. Baca ulang bagian itu atau tanyakan padaku, lalu coba kuis akhir lagi.",
		"exact_evidence":    "Semua jawaban sesuai dengan jawaban acuan.",
		"exact_feedback":    "Luar biasa — semua jawaban benar.",
		"intent_simpler":    "Jelaskan pelajaran ini dengan lebih sederhana.",
		"intent_quiz":       "Beri aku kuis latihan.",
		"intent_flashcards": "Buatkan flashcard untuk pelajaran ini.",
		"intent_examples":   "Tunjukkan contoh di dunia nyata.",
		"intent_video":      "Carikan video tentang ini.",
		"intent_image":      "Tunjukkan gambar untuk ini.",
	},
}

func tr(lang, key string, args ...any) string {
	m, ok := messages[lang]
	if !ok {
		m = messages["en"]
	}
	return fmt.Sprintf(m[key], args...)
}
