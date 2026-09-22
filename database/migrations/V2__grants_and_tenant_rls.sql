REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC;

GRANT USAGE ON SCHEMA public TO tallerflow_app;
GRANT SELECT, INSERT, UPDATE ON users, sessions TO tallerflow_app;
GRANT DELETE ON sessions TO tallerflow_app;
GRANT SELECT, INSERT, UPDATE ON workshops, memberships TO tallerflow_app;

ALTER TABLE workshops ENABLE ROW LEVEL SECURITY;
ALTER TABLE workshops FORCE ROW LEVEL SECURITY;
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships FORCE ROW LEVEL SECURITY;

CREATE POLICY workshops_tenant_isolation ON workshops
    FOR ALL
    TO tallerflow_app
    USING (id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (id = nullif(current_setting('app.workshop_id', true), '')::uuid);

CREATE POLICY memberships_tenant_isolation ON memberships
    FOR ALL
    TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);

ALTER DEFAULT PRIVILEGES FOR ROLE tallerflow_migrator IN SCHEMA public
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE tallerflow_migrator IN SCHEMA public
    REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE tallerflow_migrator IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tallerflow_app;
ALTER DEFAULT PRIVILEGES FOR ROLE tallerflow_migrator IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO tallerflow_app;

