ALTER TABLE bootstrap_state
    ADD COLUMN claimed_at timestamptz,
    ADD COLUMN workshop_id uuid REFERENCES workshops(id);
ALTER TABLE audit_events
    ADD COLUMN entity_type text,
    ADD COLUMN entity_id uuid,
    ADD COLUMN metadata jsonb NOT NULL DEFAULT '{}';
GRANT SELECT,INSERT,UPDATE ON password_reset_tokens TO tallerflow_app;
GRANT SELECT,INSERT ON audit_events TO tallerflow_app;
