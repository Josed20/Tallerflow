-- Sprint 3: Android app sessions live next to browser sessions.
-- WEB sessions use the cookie + CSRF and last eight hours; MOBILE sessions
-- use Authorization: Bearer and last 30 days, renewed with use. A token only
-- authenticates through the transport of its own kind, and a login replaces
-- only the previous session of the same kind.
ALTER TABLE user_sessions
    ADD COLUMN kind text NOT NULL DEFAULT 'WEB',
    ADD CONSTRAINT user_sessions_kind_valid CHECK (kind IN ('WEB', 'MOBILE'));

DROP INDEX user_sessions_user_active_idx;
CREATE INDEX user_sessions_user_kind_active_idx
    ON user_sessions (user_id, kind, expires_at)
    WHERE revoked_at IS NULL;
