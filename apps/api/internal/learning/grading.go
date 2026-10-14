package learning

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/curriculum"
)

// grade scores a final quiz. Answers that exactly match a reference answer
// are scored 100 here; only the rest go to the AI evaluator, and if every
// answer matches, the AI isn't called at all. Returns evaluator-shaped JSON
// (validated by assessment.Score) and who scored it.
func (s *Service) grade(ctx context.Context, userID string, cur *curriculum.Curriculum, n *curriculum.Node, lesson *curriculum.Lesson,
	st state, lang string, answers map[string]string) (json.RawMessage, string, error) {
	matched := map[string]int{}   // objective -> exact matches
	unmatched := map[string]int{} // objective -> items left for the AI
	var rest []curriculum.AssessmentItem
	for _, it := range lesson.Assessment {
		if ai.ExactMatch(answers[it.ID], it.AcceptableAnswers) {
			matched[it.ObjectiveID]++
		} else {
			unmatched[it.ObjectiveID]++
			rest = append(rest, it)
		}
	}
	if len(rest) == 0 {
		scores := map[string]int{}
		for _, o := range n.Objectives {
			scores[o.ID] = 100
		}
		raw, err := json.Marshal(map[string]any{"objectiveScores": scores, "evidence": []string{tr(lang, "exact_evidence")}, "feedback": tr(lang, "exact_feedback")})
		return raw, "exact-match", err
	}

	// Ask the evaluator only about the objectives that still need judging.
	local := cur.LocalNode(n.ID, lang)
	sub := *local
	sub.Objectives = nil
	for _, o := range local.Objectives {
		if unmatched[o.ID] > 0 {
			sub.Objectives = append(sub.Objectives, o)
		}
	}
	res, err := s.AI.Evaluate(ctx, ai.EvalInput{
		Policy: cur.Tutor.EvaluatorPolicy, Locale: lang, LanguagePolicy: cur.Tutor.LanguagePolicy[lang],
		TargetLevel: st.enrollment.TargetLevel, Node: &sub, Items: rest, Answers: answers,
		GoalHint: goalHint(cur, st.profile),
	})
	if err != nil {
		slog.Warn("evaluator call failed", "provider", s.AI.Name(), "err", err)
		return nil, "", errf(502, "evaluator_unavailable", true, "Your answers couldn't be scored right now. Nothing was saved — please submit again.")
	}
	s.recordUsage(ctx, userID, "evaluate", res.Usage)

	var partial struct {
		ObjectiveScores map[string]float64 `json:"objectiveScores"`
		Evidence        []string           `json:"evidence"`
		Feedback        string             `json:"feedback"`
	}
	if err := json.Unmarshal(res.Raw, &partial); err != nil || partial.ObjectiveScores == nil {
		slog.Warn("evaluator result rejected", "provider", s.AI.Name(), "err", err)
		return nil, "", errf(502, "evaluator_malformed", true, "Scoring returned an invalid result. Nothing was saved — please submit again.")
	}
	// Each objective's score is the item-weighted mean of exact matches (100)
	// and the evaluator's score for the remaining items. Missing or
	// out-of-range evaluator scores are passed through for assessment.Score
	// to reject.
	scores := map[string]any{}
	for _, o := range n.Objectives {
		m, u := matched[o.ID], unmatched[o.ID]
		if u == 0 {
			scores[o.ID] = 100
			continue
		}
		v, ok := partial.ObjectiveScores[o.ID]
		if !ok {
			continue
		}
		if v < 0 || v > 100 || math.IsNaN(v) {
			scores[o.ID] = v
			continue
		}
		scores[o.ID] = math.Round((100*float64(m) + v*float64(u)) / float64(m+u))
	}
	raw, err := json.Marshal(map[string]any{"objectiveScores": scores, "evidence": partial.Evidence, "feedback": partial.Feedback})
	return raw, s.AI.Name(), err
}
