// Package tutor owns the lesson state machine and the prompt contract for the
// AI tutor. The AI may *suggest* moving on; the server decides.
package tutor

type Phase string

const (
	PhaseIntroduction        Phase = "INTRODUCTION"
	PhaseExplanation         Phase = "EXPLANATION"
	PhaseGuidedPractice      Phase = "GUIDED_PRACTICE"
	PhaseIndependentPractice Phase = "INDEPENDENT_PRACTICE"
	PhaseAssessment          Phase = "ASSESSMENT"
	PhaseReview              Phase = "REVIEW"
	PhaseMastered            Phase = "MASTERED"
)

// Teaching phases accept chat messages. ASSESSMENT takes a structured
// submission instead, and MASTERED is terminal.
var chatPhases = map[Phase]bool{
	PhaseIntroduction:        true,
	PhaseExplanation:         true,
	PhaseGuidedPractice:      true,
	PhaseIndependentPractice: true,
	PhaseReview:              true,
}

func (p Phase) AcceptsChat() bool { return chatPhases[p] }

func (p Phase) Valid() bool {
	return chatPhases[p] || p == PhaseAssessment || p == PhaseMastered
}

type rule struct {
	next Phase
	// minTurns: learner replies needed in this phase before the tutor's
	// advance suggestion is honoured.
	minTurns int
	// maxTurns: after this many learner replies the server advances anyway, so
	// a learner is never stuck because the model forgot to suggest it.
	maxTurns int
}

var rules = map[Phase]rule{
	PhaseIntroduction:        {PhaseExplanation, 1, 3},
	PhaseExplanation:         {PhaseGuidedPractice, 1, 5},
	PhaseGuidedPractice:      {PhaseIndependentPractice, 2, 8},
	PhaseIndependentPractice: {PhaseAssessment, 2, 8},
	PhaseReview:              {PhaseAssessment, 2, 6},
}

// Action is the tutor's suggestion for the conversation.
type Action string

const (
	ActionContinue     Action = "continue"
	ActionAskRetry     Action = "ask_retry"
	ActionAdvancePhase Action = "advance_phase"
)

func (a Action) Valid() bool {
	return a == ActionContinue || a == ActionAskRetry || a == ActionAdvancePhase
}

// Transition returns the phase after a learner turn. turns is the number of
// learner replies in the current phase *including* the one just made.
func Transition(current Phase, turns int, suggested Action) (next Phase, advanced bool) {
	r, ok := rules[current]
	if !ok {
		return current, false
	}
	if turns >= r.maxTurns {
		return r.next, true
	}
	// Don't move on while the learner is still correcting a mistake.
	if suggested == ActionAdvancePhase && turns >= r.minTurns {
		return r.next, true
	}
	return current, false
}

// NextOf returns the phase a teaching phase leads to (or the phase itself).
func NextOf(p Phase) Phase {
	if r, ok := rules[p]; ok {
		return r.next
	}
	return p
}

// Steps is the ordered list of phases shown in the client's stepper.
var Steps = []Phase{PhaseIntroduction, PhaseExplanation, PhaseGuidedPractice, PhaseIndependentPractice, PhaseAssessment}
