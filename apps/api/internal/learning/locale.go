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
		"coming_soon":      "This lesson is coming soon.",
		"master_first":     "Master first: %s.",
		"assessment_intro": "Time for your assessment. Answer each question on your own — there are no hints here. Your answers are scored against this lesson's objectives, and you need %d/100 to master it.",
		"mastered":         "You scored %d/100 and mastered %q. %s",
		"review":           "You scored %d/100 — you need %d to master this lesson. %s Let's review: %s. Send a message when you're ready to practise.",
	},
	"id": {
		"coming_soon":      "Pelajaran ini segera hadir.",
		"master_first":     "Kuasai dulu: %s.",
		"assessment_intro": "Saatnya asesmen. Jawab setiap pertanyaan sendiri — tidak ada petunjuk di sini. Jawabanmu dinilai berdasarkan tujuan pelajaran ini, dan kamu butuh nilai %d/100 untuk menguasainya.",
		"mastered":         "Nilaimu %d/100 dan kamu menguasai %q. %s",
		"review":           "Nilaimu %d/100 — kamu butuh %d untuk menguasai pelajaran ini. %s Ayo ulas: %s. Kirim pesan kalau kamu sudah siap berlatih.",
	},
}

func tr(lang, key string, args ...any) string {
	m, ok := messages[lang]
	if !ok {
		m = messages["en"]
	}
	return fmt.Sprintf(m[key], args...)
}
