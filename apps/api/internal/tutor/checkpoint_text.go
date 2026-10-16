package tutor

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var formatTarget = regexp.MustCompile(`(?i)\b(house style|capitals|capital letters|uppercase|lowercase|upper case|lower case|capitalise|capitalize|capitalization|capitalisation|punctuation|punctuate|no comma|without (?:a )?comma|huruf besar|huruf kecil|huruf kapital|tanda baca|tanpa koma|format tanggal)\b`)

// FormatSensitiveText keeps content normalisation from deciding a score
// when the prompt explicitly assesses formatting. The grader applies its rubric.
func FormatSensitiveText(it CheckItem) bool {
	return it.Type == QuestionText && formatTarget.MatchString(it.Prompt)
}

var capitalTarget = regexp.MustCompile(`(?i)\b(capitals|capital letters|uppercase|lowercase|upper case|lower case|capitalise|capitalize|capitalization|capitalisation|huruf besar|huruf kapital)\b`)

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

// A bare response-length instruction supplies no evidence for a unique
// answer. Keep this guard narrow; semantic checks belong to the verifier.
var contextlessLetterRequest = regexp.MustCompile(`(?i)^(?:(?:tulis(?:kan)?|ketik(?:kan)?|sebutkan) (?:1|satu) huruf(?: yang (?:benar|tepat))?|(?:write|type|enter|give) (?:(?:1|one|a|the) )?(?:(?:correct|right) )?letter(?: that is (?:correct|right))?)(?: (?:jawab dalam bahasa inggris|answer in english))?$`)

func fixedGradeExplanation(score int, key, lang string, caseOnly bool) string {
	if lang == "id" {
		if caseOnly {
			return "Benar. Huruf besar dan kecil sama-sama diterima karena soal ini tidak meminta kapitalisasi tertentu."
		}
		if score == 100 {
			return "Benar. Jawabanmu sesuai dengan jawaban yang diminta."
		}
		if score == 0 {
			return fmt.Sprintf("Belum tepat. Jawaban yang benar: %s.", key)
		}
		return fmt.Sprintf("Sebagian benar (%d/100). Jawaban lengkap yang benar: %s.", score, key)
	}
	if caseOnly {
		return "Correct. Uppercase and lowercase are both accepted because this question does not require a particular case."
	}
	if score == 100 {
		return "Correct. Your answer matches the expected answer."
	}
	if score == 0 {
		return fmt.Sprintf("Not quite. The correct answer is %s.", key)
	}
	return fmt.Sprintf("Partly correct (%d/100). The complete correct answer is %s.", score, key)
}
