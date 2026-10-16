package tutor

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var formatTarget = regexp.MustCompile(`(?i)\b(house style|capitals|capital letters|capitalise|capitalize|capitalization|capitalisation|punctuation|punctuate|no comma|without (?:a )?comma|huruf besar|huruf kapital|tanda baca|tanpa koma|format tanggal)\b`)

// FormatSensitiveText keeps content normalisation from deciding a score
// when the prompt explicitly assesses formatting. The grader applies its rubric.
func FormatSensitiveText(it CheckItem) bool {
	return it.Type == QuestionText && formatTarget.MatchString(it.Prompt)
}

var capitalTarget = regexp.MustCompile(`(?i)\b(capitals|capital letters|capitalise|capitalize|capitalization|capitalisation|huruf besar|huruf kapital)\b`)

// MatchFormattedText is the conservative offline fallback: keep punctuation,
// and keep case too when capitalization itself is the target. More flexible
// valid edits are left to the live rubric grader.
func MatchFormattedText(it CheckItem, answer string) bool {
	clean := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, ref := range it.Accepted {
		if capitalTarget.MatchString(it.Prompt) {
			if clean(answer) == clean(ref) {
				return true
			}
		} else if strings.EqualFold(clean(answer), clean(ref)) {
			return true
		}
	}
	return false
}

// LetterListMatch preserves old text checkpoints without accepting an extra
// letter, missing letter, or arbitrary words. A reference must contain at
// least two separate one-letter names; spelling a word is a different task.
func LetterListMatch(answer string, accepted []string) bool {
	parse := func(s string) []string {
		words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) })
		var letters []string
		for _, w := range words {
			if w == "and" || w == "dan" {
				continue
			}
			if len(w) != 1 || w[0] < 'a' || w[0] > 'z' {
				return nil
			}
			letters = append(letters, w)
		}
		slices.Sort(letters)
		return letters
	}
	got := parse(answer)
	if len(got) < 2 {
		return false
	}
	for _, ref := range accepted {
		want := parse(ref)
		if len(want) >= 2 && slices.Equal(got, want) {
			return true
		}
	}
	return false
}
