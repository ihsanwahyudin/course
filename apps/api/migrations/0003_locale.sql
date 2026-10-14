-- UI language per user (en | id), and the language a lesson session was
-- started in, so the tutor doesn't switch language mid-conversation.
ALTER TABLE users ADD COLUMN locale TEXT NOT NULL DEFAULT 'en' CHECK (locale IN ('en', 'id'));
ALTER TABLE learning_sessions ADD COLUMN locale TEXT NOT NULL DEFAULT 'en' CHECK (locale IN ('en', 'id'));
