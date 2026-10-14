-- Lessons are now reading material with an AI helper; the guided phases are
-- gone. Open sessions continue as LEARNING with their chat kept as history.
ALTER TABLE learning_sessions DROP CONSTRAINT learning_sessions_phase_check;
UPDATE learning_sessions SET phase = 'LEARNING' WHERE phase <> 'MASTERED';
ALTER TABLE learning_sessions ADD CONSTRAINT learning_sessions_phase_check CHECK (phase IN ('LEARNING', 'MASTERED'));

-- Flashcards a learner saved, with a simple spaced-repetition schedule.
CREATE TABLE flashcards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_slug TEXT NOT NULL,
  node_id TEXT NOT NULL,
  front TEXT NOT NULL CHECK (char_length(front) BETWEEN 1 AND 160),
  back TEXT NOT NULL CHECK (char_length(back) BETWEEN 1 AND 160),
  source TEXT NOT NULL CHECK (source IN ('lesson', 'ai')),
  due_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  interval_days REAL NOT NULL DEFAULT 0,
  reps INT NOT NULL DEFAULT 0,
  lapses INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, course_slug, node_id, front)
);
CREATE INDEX flashcards_due ON flashcards (user_id, due_at);

-- Shared cache of AI output (generic quizzes, flashcards, search results)
-- so the same request isn't paid for twice.
CREATE TABLE ai_cache (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL
);

-- Tokens used per learner per day, for the daily budget.
CREATE TABLE ai_usage (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  day DATE NOT NULL,
  input_tokens BIGINT NOT NULL DEFAULT 0,
  cached_tokens BIGINT NOT NULL DEFAULT 0,
  output_tokens BIGINT NOT NULL DEFAULT 0,
  calls INT NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, day)
);
