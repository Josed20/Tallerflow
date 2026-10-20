-- Sprint 3: open workshop registration verified by email.
-- Signup itself needs no new grants: the API sets app.workshop_id to the new
-- workshop id before inserting it, so the existing tenant policies accept it.

ALTER TABLE users
    ADD COLUMN email_verified_at timestamptz,
    ADD COLUMN terms_accepted_at timestamptz,
    ADD CONSTRAINT users_email_verified_after_creation
        CHECK (email_verified_at IS NULL OR email_verified_at >= created_at),
    ADD CONSTRAINT users_terms_accepted_after_creation
        CHECK (terms_accepted_at IS NULL OR terms_accepted_at >= created_at);

-- Accounts that exist before open registration were provisioned by the team
-- (CLI bootstrap, first-workshop onboarding or an emailed invitation link), so
-- they keep signing in without a verification step.
UPDATE users SET email_verified_at = created_at WHERE email_verified_at IS NULL;

CREATE TABLE email_verification_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT email_verification_tokens_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT email_verification_tokens_used_after_creation CHECK (used_at IS NULL OR used_at >= created_at)
);

COMMENT ON COLUMN email_verification_tokens.token_hash IS 'SHA-256 of the emailed verification token; raw tokens must never be stored.';

-- A resend replaces the pending token, so at most one link per user works.
CREATE UNIQUE INDEX email_verification_tokens_one_pending_per_user_idx
    ON email_verification_tokens (user_id)
    WHERE used_at IS NULL;

GRANT SELECT, INSERT, UPDATE, DELETE ON email_verification_tokens TO tallerflow_app;
