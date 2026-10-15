-- Order attempts by insertion, not by clock: the database clock can step back
-- a little (e.g. a VM resyncing), which put a newer attempt before an older one.
ALTER TABLE assessment_attempts ADD COLUMN seq BIGSERIAL;
CREATE INDEX assessment_attempts_session_seq ON assessment_attempts (session_id, seq);
