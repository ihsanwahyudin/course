-- Lesson step actions (answers, questions, moving on) count as activity for
-- the streak and weekly numbers, now that lessons no longer store chat messages.
CREATE TABLE lesson_activity (
  id BIGSERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_id UUID NOT NULL REFERENCES learning_sessions(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX lesson_activity_user ON lesson_activity (user_id, created_at DESC);
