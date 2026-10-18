package tutor

import (
	"fmt"
	"regexp"
	"strings"
)

var recognitionWord = regexp.MustCompile(`^[a-z]+(?:'[a-z]+)?$`)

// The bank must also occur in teaching steps, not only the reading tab,
// practice questions, or checkpoint. Wording edits refresh these steps in place.
func validateRecognitionExposure(plan Plan, vocabulary []string) error {
	if len(vocabulary) == 0 {
		return nil
	}
	var teaching strings.Builder
	add := func(text string) { teaching.WriteString(" " + text) }
	for _, step := range plan.Steps {
		if step.Kind != StepConcept && step.Kind != StepExamples && step.Kind != StepArtifact {
			continue
		}
		add(step.Text)
		add(step.Body)
		for _, pair := range step.Pairs {
			add(pair.EN)
		}
		for _, example := range step.Examples {
			add(example.EN)
		}
		if sheet := step.Artifact; sheet != nil {
			add(sheet.Intro)
			for _, row := range sheet.Rows {
				add(strings.Join(row, " "))
			}
			for _, tip := range sheet.Tips {
				add(tip)
			}
			for _, example := range sheet.Examples {
				add(example.EN)
			}
		}
	}
	text := " " + NormalizeWords(teaching.String()) + " "
	for _, word := range vocabulary {
		if !strings.Contains(text, " "+NormalizeWords(word)+" ") {
			return fmt.Errorf("recognition word %q is missing from teaching steps", word)
		}
	}
	return nil
}

// ValidateRecognitionVocabulary checks single-word recognition options,
// including wrong choices, against an author's explicit bank of taught words.
// Explanatory choices and sentence tasks still require the teacher verifier;
// this is not an inferred vocabulary check over arbitrary lesson prose.
func ValidateRecognitionVocabulary(items []CheckItem, vocabulary []string) error {
	if len(vocabulary) == 0 {
		return nil
	}
	known := map[string]bool{}
	for _, word := range vocabulary {
		known[NormalizeWords(word)] = true
	}
	for _, it := range items {
		if it.Type != QuestionMCQ && it.Type != QuestionMulti {
			continue
		}
		for _, option := range it.Options {
			word := NormalizeWords(option)
			if recognitionWord.MatchString(word) && !known[word] {
				return fmt.Errorf("question %s: recognition option %q is not in the lesson's taught vocabulary; use only: %s", it.ID, option, strings.Join(vocabulary, ", "))
			}
		}
	}
	return nil
}
