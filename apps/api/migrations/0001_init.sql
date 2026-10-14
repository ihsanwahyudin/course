CREATE TABLE learners (
  id UUID PRIMARY KEY,
  display_name TEXT NOT NULL,
  native_language TEXT NOT NULL DEFAULT 'id',
  current_level TEXT NOT NULL DEFAULT 'A0',
  target_level TEXT NOT NULL DEFAULT 'B2',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE node_progress (
  learner_id UUID NOT NULL REFERENCES learners(id),
  node_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('available','in_progress','mastered')),
  mastery_score SMALLINT CHECK (mastery_score BETWEEN 0 AND 100),
  mastered_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (learner_id, node_id)
);

CREATE TABLE learning_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  learner_id UUID NOT NULL REFERENCES learners(id),
  node_id TEXT NOT NULL,
  phase TEXT NOT NULL CHECK (phase IN ('INTRODUCTION','EXPLANATION','GUIDED_PRACTICE',
    'INDEPENDENT_PRACTICE','ASSESSMENT','REVIEW','MASTERED')),
  -- learner replies in the current phase; reset on each phase change
  phase_turns INT NOT NULL DEFAULT 0,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ
);

-- At most one open session per learner and node, so start is idempotent.
CREATE UNIQUE INDEX learning_sessions_one_open
  ON learning_sessions (learner_id, node_id) WHERE ended_at IS NULL;

CREATE TABLE session_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  seq BIGINT GENERATED ALWAYS AS IDENTITY,
  session_id UUID NOT NULL REFERENCES learning_sessions(id),
  role TEXT NOT NULL CHECK (role IN ('user','assistant','system')),
  content TEXT NOT NULL,
  phase TEXT NOT NULL,
  -- tutor action / observed mistake tags; advisory only
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX session_messages_session_seq ON session_messages (session_id, seq);

CREATE TABLE assessment_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES learning_sessions(id),
  score SMALLINT NOT NULL CHECK (score BETWEEN 0 AND 100),
  threshold SMALLINT NOT NULL,
  passed BOOLEAN NOT NULL,
  objective_scores JSONB NOT NULL,
  evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
  feedback TEXT NOT NULL,
  answers JSONB NOT NULL,
  -- raw evaluator output, kept for auditability
  evaluator_response JSONB NOT NULL,
  evaluator TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX assessment_attempts_session ON assessment_attempts (session_id, created_at);
