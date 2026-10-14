-- Lessons now ship with hand-written steps. Open sessions still on an
-- AI-written plan are closed (kept, including the learner's questions to the
-- tutor) so the next visit starts the lesson's hand-written steps. Progress
-- and mastery are unaffected.
UPDATE learning_sessions SET ended_at = NOW() WHERE ended_at IS NULL AND plan IS NOT NULL;
