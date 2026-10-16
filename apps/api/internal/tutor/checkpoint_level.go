package tutor

import (
	"fmt"
	"regexp"
	"strings"
)

var spellingText = regexp.MustCompile(`(?i)\b(spell|spelling|eja|mengeja)\b.*\b(dash|hyphen|strip|letters|huruf)\b`)
var listText = regexp.MustCompile(`(?i)\b(list|write|name|tulis|sebutkan)\b.*\b(vowels|vowel letters|huruf.{0,12}vokal|words.{0,25}capital|kata.{0,25}kapital)\b`)

// ValidateLevelQuestions applies the same interaction rules to AI drafts and
// authored fallbacks. It does not guess whether vocabulary has been taught;
// that requires the content audit and the teacher verifier.
func ValidateLevelQuestions(items []CheckItem, level string) error {
	for _, it := range items {
		if it.Type == QuestionLetters {
			maxExtra := maxDecoys
			switch level {
			case "A0":
				maxExtra = 0
			case "A1":
				maxExtra = 2
			}
			if len(it.Tokens)-len(letters(it.Solution)) > maxExtra {
				return fmt.Errorf("%s: level %s allows at most %d extra letter tiles", it.ID, level, maxExtra)
			}
		}
		if it.Type == QuestionText && (spellingText.MatchString(it.Prompt) || listText.MatchString(it.Prompt)) {
			return fmt.Errorf("%s: use letters for spelling or multi for a list, rather than formatted text", it.ID)
		}
		if level == "A0" && it.Type == QuestionText {
			for _, answer := range it.Accepted {
				if len(strings.Fields(answer)) > 3 {
					return fmt.Errorf("%s: A0 text answers must be a taught word or phrase of at most three words", it.ID)
				}
			}
		}
	}
	return nil
}
