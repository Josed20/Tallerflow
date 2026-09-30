CREATE UNIQUE INDEX workshop_members_one_workshop_per_user_idx ON workshop_members(user_id);
CREATE TABLE team_invitations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    email text NOT NULL CHECK (email = lower(btrim(email))),
    role text NOT NULL CHECK (role IN ('ADMIN','OPERATOR')),
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX team_invitation_active_email_idx ON team_invitations(workshop_id,email) WHERE consumed_at IS NULL;
ALTER TABLE team_invitations ENABLE ROW LEVEL SECURITY; ALTER TABLE team_invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY invitations_tenant ON team_invitations TO tallerflow_app USING (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid) WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid);
GRANT SELECT,INSERT,UPDATE ON team_invitations TO tallerflow_app;
GRANT SELECT,UPDATE ON team_invitations TO tallerflow_bootstrap;
GRANT SELECT ON users, workshop_members TO tallerflow_bootstrap;
CREATE POLICY invitations_bootstrap_consume ON team_invitations TO tallerflow_bootstrap
    USING (true) WITH CHECK (true);
