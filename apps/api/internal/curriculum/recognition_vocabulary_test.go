package curriculum

import "testing"

func TestRecognitionVocabularyRequiresTeachingMaterial(t *testing.T) {
	l := &Lesson{NodeID: "sounds", RecognitionVocabulary: []string{"leave"}, TeachingNotes: []string{"leave"}, Material: []Section{{Blocks: []Block{
		{Type: BlockCheck, Prompt: "leave", Options: []string{"leave", "live"}},
	}}}}
	if len(validateRecognitionVocabulary(l)) == 0 {
		t.Fatal("teacher notes and quiz choices are not learner teaching")
	}
	l.Material[0].Blocks = append(l.Material[0].Blocks, Block{Type: BlockExamples, Rows: []ExampleRow{{EN: "leave /iː/", ID: "pergi · ii panjang"}}})
	if errs := validateRecognitionVocabulary(l); len(errs) > 0 {
		t.Fatal(errs)
	}
	for _, bank := range [][]string{{"leave", "leave"}, {"Leave"}, {"two words"}, {"leave", "safe"}} {
		l.RecognitionVocabulary = bank
		if len(validateRecognitionVocabulary(l)) == 0 {
			t.Fatalf("invalid or untaught bank accepted: %v", bank)
		}
	}
}
