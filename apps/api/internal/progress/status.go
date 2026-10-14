// Package progress derives learner-facing node statuses from the curriculum and
// stored progress rows. Locked status is always derived from prerequisites and
// never stored.
package progress

import "fluentpath/api/internal/curriculum"

type Status string

const (
	StatusMastered   Status = "mastered"
	StatusInProgress Status = "in_progress"
	StatusAvailable  Status = "available"
	StatusLocked     Status = "locked"
	StatusComingSoon Status = "coming_soon"
)

// Stored is the persisted state of one node for a learner.
type Stored struct {
	Status       string // available | in_progress | mastered
	MasteryScore *int
}

// Derive computes the status of every node.
func Derive(c *curriculum.Curriculum, stored map[string]Stored) map[string]Status {
	out := make(map[string]Status, len(c.Nodes))
	for _, n := range c.Nodes {
		out[n.ID] = statusOf(n, stored)
	}
	return out
}

func statusOf(n *curriculum.Node, stored map[string]Stored) Status {
	s, has := stored[n.ID]
	if has && s.Status == string(StatusMastered) {
		return StatusMastered
	}
	if !n.Ready() {
		return StatusComingSoon
	}
	for _, p := range n.PrerequisiteIDs {
		if stored[p].Status != string(StatusMastered) {
			return StatusLocked
		}
	}
	if has && s.Status == string(StatusInProgress) {
		return StatusInProgress
	}
	return StatusAvailable
}

// CanStart reports whether a learner may open a lesson session on the node.
// Mastered nodes may be revisited for practice; that never changes mastery.
func CanStart(st Status) bool {
	return st == StatusAvailable || st == StatusInProgress || st == StatusMastered
}

// NextSuggested picks the first in-progress node, else the first available
// node, in curriculum (topological) order.
func NextSuggested(c *curriculum.Curriculum, statuses map[string]Status) *curriculum.Node {
	for _, want := range []Status{StatusInProgress, StatusAvailable} {
		for _, n := range c.Nodes {
			if statuses[n.ID] == want {
				return n
			}
		}
	}
	return nil
}

// CurrentLevel is the level of the suggested next node; if nothing is left to
// start, it is the highest level with a mastered node (or A0).
func CurrentLevel(c *curriculum.Curriculum, statuses map[string]Status) string {
	if n := NextSuggested(c, statuses); n != nil {
		return n.Level
	}
	best := "A0"
	for _, n := range c.Nodes {
		if statuses[n.ID] == StatusMastered && curriculum.LevelIndex(n.Level) > curriculum.LevelIndex(best) {
			best = n.Level
		}
	}
	return best
}

// NewlyUnlocked returns nodes that are startable in after but were locked in
// before, in curriculum order.
func NewlyUnlocked(c *curriculum.Curriculum, before, after map[string]Status) []string {
	var ids []string
	for _, n := range c.Nodes {
		if before[n.ID] == StatusLocked && (after[n.ID] == StatusAvailable || after[n.ID] == StatusInProgress) {
			ids = append(ids, n.ID)
		}
	}
	return ids
}
