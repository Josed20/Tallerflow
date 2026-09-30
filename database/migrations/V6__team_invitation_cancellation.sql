ALTER TABLE team_invitations
    ADD COLUMN canceled_at timestamptz;

DROP INDEX team_invitations_active_email_idx;

CREATE UNIQUE INDEX team_invitations_active_email_idx
    ON team_invitations (workshop_id, email)
    WHERE consumed_at IS NULL AND canceled_at IS NULL;

CREATE INDEX team_invitations_pending_idx
    ON team_invitations (workshop_id, expires_at)
    WHERE consumed_at IS NULL AND canceled_at IS NULL;
