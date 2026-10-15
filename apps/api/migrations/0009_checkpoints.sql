-- Checkpoints are written fresh for every attempt (by the AI, or from the
-- lesson's hand-written questions as a fallback). A checkpoint keeps its
-- questions and answer keys on the server; at most one per session is open.
CREATE TABLE checkpoints (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES learning_sessions(id),
  -- practice: taken after the lesson was mastered; it never changes progress
  practice BOOLEAN NOT NULL DEFAULT FALSE,
  locale TEXT NOT NULL,
  source TEXT NOT NULL,
  items JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  submitted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX checkpoints_one_open ON checkpoints (session_id) WHERE submitted_at IS NULL;
CREATE INDEX checkpoints_session ON checkpoints (session_id, created_at);

-- Every graded answer with its explanation, so old attempts can be reviewed.
ALTER TABLE assessment_attempts
  ADD COLUMN checkpoint_id UUID REFERENCES checkpoints(id),
  ADD COLUMN practice BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN review JSONB;
