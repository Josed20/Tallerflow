CREATE TABLE team_invitations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops (id) ON DELETE CASCADE,
    email citext NOT NULL,
    role text NOT NULL,
    token_hash bytea NOT NULL UNIQUE,
    created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    consumed_by uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT team_invitations_role_valid CHECK (role IN ('ADMIN', 'OPERATOR')),
    CONSTRAINT team_invitations_email_not_blank CHECK (length(btrim(email::text)) BETWEEN 3 AND 320),
    CONSTRAINT team_invitations_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT team_invitations_consumed_after_creation CHECK (consumed_at IS NULL OR consumed_at >= created_at)
);

CREATE UNIQUE INDEX team_invitations_active_email_idx
    ON team_invitations (workshop_id, email)
    WHERE consumed_at IS NULL;

CREATE INDEX team_invitations_workshop_created_idx
    ON team_invitations (workshop_id, created_at DESC);

CREATE INDEX team_invitations_token_active_idx
    ON team_invitations (token_hash)
    WHERE consumed_at IS NULL;

CREATE UNIQUE INDEX workshop_members_one_workshop_per_user_idx
    ON workshop_members (user_id);

GRANT SELECT, INSERT, UPDATE ON team_invitations TO tallerflow_app;

ALTER TABLE team_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_invitations FORCE ROW LEVEL SECURITY;

CREATE POLICY team_invitations_tenant_isolation ON team_invitations
    FOR ALL TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);

CREATE POLICY team_invitations_public_consume_lookup ON team_invitations
    FOR SELECT TO tallerflow_app
    USING (
        token_hash = decode(nullif(current_setting('app.invitation_token_hash', true), ''), 'hex')
    );

CREATE POLICY team_invitations_public_consume_update ON team_invitations
    FOR UPDATE TO tallerflow_app
    USING (
        token_hash = decode(nullif(current_setting('app.invitation_token_hash', true), ''), 'hex')
    )
    WITH CHECK (
        token_hash = decode(nullif(current_setting('app.invitation_token_hash', true), ''), 'hex')
    );
