package tutor

import (
	"fmt"
	"regexp"
	"strings"
)

var soundComparisonTopic = regexp.MustCompile(`(?i)\b(?:bunyi|vokal|pengucapan|diucapkan|sounds?|vowels?|pronunciation|pronounced|rhymes?|berima)\b`)

// Bare "as a capital letter" or "would like a short answer" are not comparisons.
// General as/like cues therefore require a quoted reference; unquoted references
// need a more specific cue such as "as in" or "sound like".
var quotedSoundReference = regexp.MustCompile("(?i)\\b(?:seperti(?:\\s+(?:pada|di|dalam))?|(?:as|like)(?:\\s+in)?|rhymes?\\s+with|berima\\s+dengan)\\s+(?:(?:the\\s+word|word|kata)\\s+)?['\"‘’“”*`]([a-z]+)['\"‘’“”*`](?:$|[^a-z])")
var plainSoundReference = regexp.MustCompile(`(?i)\b(?:seperti(?:\s+(?:pada|di|dalam))?|as\s+in|(?:sounds?|vowels?)\s+(?:as|like)|like\s+in|rhymes?\s+with|berima\s+dengan)\s+(?:(?:the\s+word|word|kata)\s+)?([a-z]+)\b`)

// A sound-matching question needs a different example word as its answer.
// This deliberately checks reference cues, not every word in the prompt:
// listing choices or quoting a sentence to analyse is legitimate.
func validateSoundComparison(prompt string, answers []string) error {
	if !soundComparisonTopic.MatchString(prompt) {
		return nil
	}
	refs := quotedSoundReference.FindAllStringSubmatch(prompt, -1)
	refs = append(refs, plainSoundReference.FindAllStringSubmatch(prompt, -1)...)
	for _, ref := range refs {
		word := strings.ToLower(ref[1])
		for _, answer := range answers {
			if NormalizeWords(answer) == word {
				return fmt.Errorf("a sound-comparison answer must differ from reference word %q; use another taught word or a listening question without showing its answer", word)
			}
		}
	}
	return nil
}
