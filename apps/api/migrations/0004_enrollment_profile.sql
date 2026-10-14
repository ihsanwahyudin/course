-- Onboarding answers per course enrollment. profile maps question id ->
-- option id (plus "<question>_text" for free text). onboarded_at stays NULL
-- until the learner answers or skips, so existing learners are offered it.
ALTER TABLE enrollments
  ADD COLUMN profile JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN profile_version INT NOT NULL DEFAULT 0,
  ADD COLUMN onboarded_at TIMESTAMPTZ;
