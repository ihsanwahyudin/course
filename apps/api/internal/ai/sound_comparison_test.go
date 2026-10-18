package ai

import (
	"strings"
	"testing"
)

func TestSoundComparisonWriterAndVerifierRules(t *testing.T) {
	in, _ := lessonInput(t, "a0-sounds-indonesian", "id")
	prefix := lessonPrefix(in)
	if !strings.Contains(prefix, "reference word is a comparison, never the answer itself") || !strings.Contains(prefix, "ANOTHER taught word") || !strings.Contains(prefix, "exclude ship") {
		t.Fatal("writer, similar questions and level checkpoints need the comparison rule")
	}
	verify := verifyTask(VerifyInput{LessonInput: in, Items: AuthoredCheckpoint(in.Lesson, "id")})
	if !strings.Contains(verify, `with key "ship"`) || !strings.Contains(verify, "another taught word") {
		t.Fatal("verifier needs to reject reference words used as keys")
	}
}
