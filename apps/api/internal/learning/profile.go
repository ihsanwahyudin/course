package learning

import (
	"context"

	"github.com/jackc/pgx/v5"

	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/storage"
)

// ProfileView is a course's onboarding questions in the UI language plus the
// learner's current answers.
type ProfileView struct {
	Course      CourseRef             `json:"course"`
	Questions   []curriculum.Question `json:"questions"`
	Answers     map[string]string     `json:"answers"`
	Completed   bool                  `json:"completed"`
	TargetLevel string                `json:"targetLevel"`
	Levels      []string              `json:"levels"`
}

// ProfileInput saves onboarding answers. Skip records that the learner
// declined onboarding, so it isn't offered again.
type ProfileInput struct {
	Answers     map[string]string `json:"answers"`
	TargetLevel string            `json:"targetLevel"`
	Skip        bool              `json:"skip"`
}

func (s *Service) GetProfile(ctx context.Context, userID, courseSlug string) (ProfileView, error) {
	cur, err := s.course(courseSlug)
	if err != nil {
		return ProfileView{}, err
	}
	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return ProfileView{}, err
	}
	answers := map[string]string{}
	for k, v := range st.profile {
		answers[k] = v
	}
	return ProfileView{
		Course: CourseRef{cur.Slug, cur.Text(st.lang).Title}, Questions: cur.LocalQuestions(st.lang),
		Answers: answers, Completed: st.enrollment.OnboardedAt != nil,
		TargetLevel: st.enrollment.TargetLevel, Levels: cur.LevelIDs,
	}, nil
}

// SaveProfile validates and stores onboarding answers, enrolling the learner
// if needed. The goal sets the target level unless the learner picked one.
func (s *Service) SaveProfile(ctx context.Context, userID, courseSlug string, in ProfileInput) (RoadmapView, error) {
	cur, err := s.course(courseSlug)
	if err != nil {
		return RoadmapView{}, err
	}
	if cur.Status != curriculum.CoursePublished {
		return RoadmapView{}, errf(409, "course_coming_soon", false, "%s isn't open yet.", cur.Title)
	}
	if cur.Onboarding == nil {
		return RoadmapView{}, errf(404, "no_onboarding", false, "%s has no onboarding questions.", cur.Title)
	}
	st, err := s.load(ctx, s.Store.Pool, userID, cur)
	if err != nil {
		return RoadmapView{}, err
	}

	profile, target := curriculum.Profile{}, st.enrollment.TargetLevel
	switch {
	case in.Skip && st.enrollment.OnboardedAt != nil:
		// Already answered or skipped: keep what's there.
		return s.Roadmap(ctx, userID, courseSlug)
	case !in.Skip:
		if profile, err = cur.ValidateProfile(in.Answers); err != nil {
			return RoadmapView{}, errf(400, "invalid_profile", false, "%s", err)
		}
		if o := cur.Option(profile, curriculum.GoalQuestion); o != nil && o.TargetLevel != "" {
			target = o.TargetLevel
		}
		if in.TargetLevel != "" {
			if cur.LevelIndex(in.TargetLevel) < 0 {
				return RoadmapView{}, errf(400, "invalid_profile", false, "%q is not a level of this course", in.TargetLevel)
			}
			target = in.TargetLevel
		}
	}

	err = pgx.BeginFunc(ctx, s.Store.Pool, func(tx pgx.Tx) error {
		if _, err := storage.EnsureEnrollment(ctx, tx, userID, cur.Slug, cur.LevelIDs[0], cur.TargetLevel); err != nil {
			return err
		}
		return storage.SaveProfile(ctx, tx, userID, cur.Slug, profile, cur.Onboarding.Version, target)
	})
	if err != nil {
		return RoadmapView{}, err
	}
	return s.Roadmap(ctx, userID, courseSlug)
}

// neighbours are the previous and next lessons with content, in roadmap order.
func neighbours(st state, n *curriculum.Node) (prev, next *NodeView) {
	var ready []*curriculum.Node
	for _, c := range st.cur.Nodes {
		if c.Ready() {
			ready = append(ready, c)
		}
	}
	for i, c := range ready {
		if c.ID != n.ID {
			continue
		}
		if i > 0 {
			v := nodeView(st, ready[i-1])
			prev = &v
		}
		if i+1 < len(ready) {
			v := nodeView(st, ready[i+1])
			next = &v
		}
	}
	return prev, next
}

// goalHint lets the evaluator relate feedback to the learner's goal. It is
// not used for scoring.
func goalHint(cur *curriculum.Curriculum, p curriculum.Profile) string {
	if o := cur.Option(p, curriculum.GoalQuestion); o != nil {
		return o.TutorHint
	}
	return ""
}
