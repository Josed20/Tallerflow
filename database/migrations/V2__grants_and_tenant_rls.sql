REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC;

GRANT USAGE ON SCHEMA public TO tallerflow_app, tallerflow_bootstrap;

-- The API gets only the tables it needs. It never receives ownership or a
-- bypass-RLS capability; future migrations must grant each new table explicitly.
GRANT SELECT, INSERT, UPDATE ON users, user_credentials, user_sessions, login_attempts, password_reset_tokens TO tallerflow_app;
GRANT DELETE ON user_sessions, password_reset_tokens TO tallerflow_app;
GRANT SELECT, INSERT, UPDATE ON workshops, workshop_members, audit_events TO tallerflow_app;

-- The bootstrap connection is a short-lived, separate credential. It can
-- create exactly the initial identity graph but has no DDL rights.
GRANT SELECT, INSERT, UPDATE ON bootstrap_state TO tallerflow_bootstrap;
GRANT SELECT, INSERT ON users, workshops TO tallerflow_bootstrap;
GRANT INSERT ON user_credentials, workshop_members, audit_events TO tallerflow_bootstrap;

ALTER TABLE workshops ENABLE ROW LEVEL SECURITY;
ALTER TABLE workshops FORCE ROW LEVEL SECURITY;
ALTER TABLE workshop_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE workshop_members FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY workshops_tenant_isolation ON workshops
    FOR ALL TO tallerflow_app
    USING (id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (id = nullif(current_setting('app.workshop_id', true), '')::uuid);

CREATE POLICY workshop_members_tenant_isolation ON workshop_members
    FOR ALL TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);

CREATE POLICY audit_events_tenant_isolation ON audit_events
    FOR ALL TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);

-- A bootstrap role is explicit rather than a database owner. It has no
-- BYPASSRLS attribute and may write only while a deliberate bootstrap runs.
CREATE POLICY workshops_bootstrap_insert ON workshops
    FOR INSERT TO tallerflow_bootstrap WITH CHECK (true);
-- PostgreSQL requires a matching SELECT policy for INSERT ... RETURNING,
-- which the controlled bootstrap transaction uses to obtain the new workshop id.
CREATE POLICY workshops_bootstrap_select ON workshops
    FOR SELECT TO tallerflow_bootstrap USING (true);
CREATE POLICY workshop_members_bootstrap_insert ON workshop_members
    FOR INSERT TO tallerflow_bootstrap WITH CHECK (true);
CREATE POLICY audit_events_bootstrap_insert ON audit_events
    FOR INSERT TO tallerflow_bootstrap WITH CHECK (true);

-- Login must resolve memberships before a tenant context exists. The function
-- executes under the migration role, which has a narrow select policy; the app
-- receives only the resulting active memberships.
CREATE POLICY workshop_members_migrator_lookup ON workshop_members
    FOR SELECT TO tallerflow_migrator USING (true);

CREATE FUNCTION resolve_active_memberships(p_user_id uuid)
RETURNS TABLE (workshop_id uuid, role text)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT wm.workshop_id, wm.role
    FROM workshop_members AS wm
    WHERE wm.user_id = p_user_id
      AND wm.status = 'ACTIVE'
    ORDER BY wm.created_at ASC;
$$;

REVOKE ALL ON FUNCTION resolve_active_memberships(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION resolve_active_memberships(uuid) TO tallerflow_app;

ALTER DEFAULT PRIVILEGES FOR ROLE tallerflow_migrator IN SCHEMA public
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE tallerflow_migrator IN SCHEMA public
    REVOKE ALL ON SEQUENCES FROM PUBLIC;
