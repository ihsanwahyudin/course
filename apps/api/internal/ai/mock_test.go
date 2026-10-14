package ai

import "testing"

func TestMatches(t *testing.T) {
	cases := []struct {
		answer   string
		expected []string
		want     bool
	}{
		{"She is.", []string{"is"}, true},
		{"C-A-T", []string{"c a t"}, true},
		{"an", []string{"a"}, false},
		{"They aren’t at school!", []string{"they aren't at school"}, true},
		{"", []string{"x"}, false},
		{"I get up at 6 every day", []string{"i get up at"}, true},
	}
	for _, c := range cases {
		if got := Matches(c.answer, c.expected); got != c.want {
			t.Errorf("Matches(%q, %v) = %v, want %v", c.answer, c.expected, got, c.want)
		}
	}
}
