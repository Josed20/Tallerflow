-- Identity and tenancy contract shared by the Sprint 1 services.
-- Extensions are installed by infra/postgres/init-roles.sh before Flyway runs.

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email citext NOT NULL UNIQUE,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_not_blank CHECK (length(btrim(email::text)) BETWEEN 3 AND 320),
    CONSTRAINT users_name_not_blank CHECK (length(btrim(name)) BETWEEN 1 AND 160),
    CONSTRAINT users_status_valid CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE TABLE user_credentials (
    user_id uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    password_hash text NOT NULL,
    must_change_password boolean NOT NULL DEFAULT true,
    password_changed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_credentials_password_hash_not_blank CHECK (length(password_hash) > 0)
);

CREATE TABLE workshops (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    timezone text NOT NULL DEFAULT 'America/Lima',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workshops_name_not_blank CHECK (length(btrim(name)) BETWEEN 1 AND 160),
    CONSTRAINT workshops_timezone_not_blank CHECK (length(btrim(timezone)) BETWEEN 1 AND 80)
);

CREATE TABLE workshop_members (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops (id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role text NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workshop_members_role_valid CHECK (role IN ('OWNER', 'ADMIN', 'SUPERVISOR', 'OPERATOR')),
    CONSTRAINT workshop_members_status_valid CHECK (status IN ('ACTIVE', 'INACTIVE')),
    CONSTRAINT workshop_members_workshop_user_unique UNIQUE (workshop_id, user_id)
);

CREATE INDEX workshop_members_user_id_idx ON workshop_members (user_id);
CREATE INDEX workshop_members_workshop_role_idx ON workshop_members (workshop_id, role) WHERE status = 'ACTIVE';

CREATE TABLE user_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    csrf_token_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    ip_prefix text,
    user_agent text,
    CONSTRAINT user_sessions_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT user_sessions_revoked_after_creation CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX user_sessions_user_active_idx ON user_sessions (user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX user_sessions_expiry_idx ON user_sessions (expires_at);

COMMENT ON COLUMN user_sessions.token_hash IS 'HMAC-SHA-256 of the opaque session token; raw tokens must never be stored.';
COMMENT ON COLUMN user_sessions.csrf_token_hash IS 'SHA-256 of the CSRF token; raw tokens must never be stored.';

CREATE TABLE login_attempts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email citext NOT NULL,
    ip_prefix text,
    succeeded boolean NOT NULL,
    attempted_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX login_attempts_email_attempted_at_idx ON login_attempts (email, attempted_at DESC);

CREATE TABLE password_reset_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT password_reset_tokens_expiry_after_creation CHECK (expires_at > created_at)
);

CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);

CREATE TABLE audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops (id) ON DELETE RESTRICT,
    actor_user_id uuid REFERENCES users (id) ON DELETE SET NULL,
    event_type text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_events_event_type_not_blank CHECK (length(btrim(event_type)) BETWEEN 1 AND 120)
);

CREATE INDEX audit_events_workshop_created_at_idx ON audit_events (workshop_id, created_at DESC);

-- A singleton is reserved by the bootstrap transaction before creating the
-- initial OWNER. It prevents a second workshop owner from being provisioned.
CREATE TABLE bootstrap_state (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    owner_user_id uuid NOT NULL UNIQUE REFERENCES users (id) ON DELETE RESTRICT,
    bootstrapped_at timestamptz NOT NULL DEFAULT now()
);
