-- Lessons run as an AI-written plan of steps shown one at a time.
ALTER TABLE learning_sessions
  ADD COLUMN plan JSONB,
  ADD COLUMN step_index INT NOT NULL DEFAULT 0,
  ADD COLUMN step_state JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN summary JSONB;

-- Flashcards can now also come from the learner's own mistakes.
ALTER TABLE flashcards DROP CONSTRAINT flashcards_source_check;
ALTER TABLE flashcards ADD CONSTRAINT flashcards_source_check CHECK (source IN ('lesson', 'ai', 'mistake'));

-- What the tutor remembers about a learner (visible and editable in Account).
CREATE TABLE learner_memory (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_slug TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('pattern', 'personal')),
  text TEXT NOT NULL CHECK (char_length(text) BETWEEN 1 AND 200),
  source_node TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX learner_memory_user ON learner_memory (user_id, created_at DESC);
ALTER TABLE users ADD COLUMN memory_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- Saved artifacts ("Catatanku").
CREATE TABLE notes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_slug TEXT NOT NULL,
  node_id TEXT NOT NULL,
  title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
  content JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, course_slug, node_id, title)
);
