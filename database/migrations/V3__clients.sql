CREATE TABLE clients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
    email text CHECK (email IS NULL OR email = lower(btrim(email))),
    phone text,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX clients_workshop_name_idx ON clients(workshop_id, lower(name), id);
ALTER TABLE clients ENABLE ROW LEVEL SECURITY;
ALTER TABLE clients FORCE ROW LEVEL SECURITY;
CREATE POLICY clients_tenant_isolation ON clients TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);
GRANT SELECT, INSERT, UPDATE ON clients TO tallerflow_app;
