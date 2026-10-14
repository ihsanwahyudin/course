package tutor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ErrMalformed marks provider output that failed validation.
var ErrMalformed = errors.New("malformed tutor reply")

const maxReplyRunes = 2000

type Reply struct {
	Message          string   `json:"message"`
	Action           Action   `json:"action"`
	ObservedMistakes []string `json:"observedMistakes"`
}

var tagRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_:-]{0,39}$`)

// ParseReply validates a provider's raw JSON tutor reply.
func ParseReply(raw []byte) (Reply, error) {
	var r Reply
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&r); err != nil {
		return Reply{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	r.Message = strings.TrimSpace(r.Message)
	if r.Message == "" {
		return Reply{}, fmt.Errorf("%w: empty message", ErrMalformed)
	}
	if utf8.RuneCountInString(r.Message) > maxReplyRunes {
		return Reply{}, fmt.Errorf("%w: message too long", ErrMalformed)
	}
	if !r.Action.Valid() {
		return Reply{}, fmt.Errorf("%w: unknown action %q", ErrMalformed, r.Action)
	}
	// Mistake tags are advisory; keep only well-formed ones.
	tags := make([]string, 0, len(r.ObservedMistakes))
	for _, t := range r.ObservedMistakes {
		t = strings.ToLower(strings.TrimSpace(t))
		if tagRe.MatchString(t) && len(tags) < 5 {
			tags = append(tags, t)
		}
	}
	r.ObservedMistakes = tags
	return r, nil
}
