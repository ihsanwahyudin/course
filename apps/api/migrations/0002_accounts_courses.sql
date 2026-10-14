-- Accounts and multiple courses. Existing rows are kept: the old demo learner
-- becomes a user without credentials (it can't log in) and its progress is
-- attached to the "english" course.

ALTER TABLE learners RENAME TO users;
ALTER TABLE users ADD COLUMN email TEXT CHECK (email = lower(email));
ALTER TABLE users ADD COLUMN password_hash TEXT;
CREATE UNIQUE INDEX users_email_key ON users (email) WHERE email IS NOT NULL;

-- Per-course placement replaces the single current/target level on the learner.
CREATE TABLE enrollments (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_slug TEXT NOT NULL,
  current_level TEXT NOT NULL,
  target_level TEXT NOT NULL,
  enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, course_slug)
);

INSERT INTO enrollments (user_id, course_slug, current_level, target_level, enrolled_at)
SELECT u.id, 'english', u.current_level, u.target_level, u.created_at
FROM users u
WHERE EXISTS (SELECT 1 FROM node_progress p WHERE p.learner_id = u.id)
   OR EXISTS (SELECT 1 FROM learning_sessions s WHERE s.learner_id = u.id);

ALTER TABLE users DROP COLUMN current_level;
ALTER TABLE users DROP COLUMN target_level;

ALTER TABLE node_progress RENAME COLUMN learner_id TO user_id;
ALTER TABLE node_progress ADD COLUMN course_slug TEXT NOT NULL DEFAULT 'english';
ALTER TABLE node_progress ALTER COLUMN course_slug DROP DEFAULT;
ALTER TABLE node_progress DROP CONSTRAINT node_progress_pkey;
ALTER TABLE node_progress ADD PRIMARY KEY (user_id, course_slug, node_id);

ALTER TABLE learning_sessions RENAME COLUMN learner_id TO user_id;
ALTER TABLE learning_sessions ADD COLUMN course_slug TEXT NOT NULL DEFAULT 'english';
ALTER TABLE learning_sessions ALTER COLUMN course_slug DROP DEFAULT;
DROP INDEX learning_sessions_one_open;
CREATE UNIQUE INDEX learning_sessions_one_open
  ON learning_sessions (user_id, course_slug, node_id) WHERE ended_at IS NULL;
CREATE INDEX learning_sessions_user_updated ON learning_sessions (user_id, updated_at DESC);

-- Login sessions. Only a SHA-256 of the bearer token is stored.
CREATE TABLE auth_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  user_agent TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX auth_sessions_user ON auth_sessions (user_id);
