ALTER TABLE workshops
    ADD COLUMN status text NOT NULL DEFAULT 'ACTIVE',
    ADD COLUMN status_reason text,
    ADD COLUMN status_changed_at timestamptz,
    ADD COLUMN deletion_scheduled_for timestamptz,
    ADD CONSTRAINT workshops_status_valid CHECK (status IN ('ACTIVE', 'SUSPENDED', 'PENDING_DELETION'));

CREATE TABLE console_admins (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email citext NOT NULL UNIQUE,
    password_hash text NOT NULL,
    totp_secret_ciphertext bytea NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE console_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id uuid NOT NULL REFERENCES console_admins(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    csrf_token_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX console_sessions_active_idx ON console_sessions(admin_id, expires_at) WHERE revoked_at IS NULL;

CREATE TABLE console_audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_admin_id uuid REFERENCES console_admins(id) ON DELETE SET NULL,
    workshop_id uuid,
    event_type text NOT NULL,
    reason text,
    source_ip text,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

GRANT USAGE ON SCHEMA public TO tallerflow_console;
REVOKE ALL ON console_audit_events FROM PUBLIC;
GRANT SELECT, INSERT ON console_admins, console_sessions, console_audit_events TO tallerflow_console;
GRANT UPDATE (last_seen_at, revoked_at) ON console_sessions TO tallerflow_console;

-- The SECURITY DEFINER functions are owned by the migration role. FORCE RLS
-- remains enabled, so grant that role only the cross-tenant operations used by
-- these reviewed functions; the console login itself still has no table access.
CREATE POLICY workshops_console_function_policy ON workshops TO tallerflow_migrator USING (true) WITH CHECK (true);
CREATE POLICY members_console_function_policy ON workshop_members FOR SELECT TO tallerflow_migrator USING (true);
CREATE POLICY users_console_function_policy ON users FOR SELECT TO tallerflow_migrator USING (true);
CREATE POLICY sessions_console_function_select ON user_sessions FOR SELECT TO tallerflow_migrator USING (true);
CREATE POLICY sessions_console_function_update ON user_sessions FOR UPDATE TO tallerflow_migrator USING (true) WITH CHECK (true);
CREATE POLICY orders_console_function_policy ON orders FOR SELECT TO tallerflow_migrator USING (true);
CREATE POLICY photos_console_function_policy ON stage_photos FOR SELECT TO tallerflow_migrator USING (true);
CREATE POLICY audit_console_function_delete ON audit_events FOR DELETE TO tallerflow_migrator USING (true);

CREATE OR REPLACE FUNCTION console_list_workshops(search_text text, row_limit integer, row_offset integer)
RETURNS TABLE (
    id uuid, name text, owner_email citext, created_at timestamptz,
    last_activity timestamptz, members bigint, active_orders bigint,
    photo_bytes bigint, status text, deletion_scheduled_for timestamptz
)
LANGUAGE sql SECURITY DEFINER SET search_path = public, pg_temp AS $$
    SELECT w.id, w.name, owner.email, w.created_at,
           greatest(activity.last_session, activity.last_order),
           metrics.members, metrics.active_orders, metrics.photo_bytes,
           w.status, w.deletion_scheduled_for
    FROM workshops w
    LEFT JOIN LATERAL (
        SELECT u.email FROM workshop_members wm JOIN users u ON u.id = wm.user_id
        WHERE wm.workshop_id = w.id AND wm.role = 'OWNER' AND wm.status = 'ACTIVE'
        ORDER BY wm.created_at LIMIT 1
    ) owner ON true
    LEFT JOIN LATERAL (
        SELECT max(us.last_seen_at) AS last_session, max(o.updated_at) AS last_order
        FROM workshop_members wm
        LEFT JOIN user_sessions us ON us.user_id = wm.user_id
        LEFT JOIN orders o ON o.workshop_id = w.id
        WHERE wm.workshop_id = w.id
    ) activity ON true
    CROSS JOIN LATERAL (
        SELECT
          (SELECT count(*) FROM workshop_members wm WHERE wm.workshop_id = w.id AND wm.status = 'ACTIVE') AS members,
          (SELECT count(*) FROM orders o WHERE o.workshop_id = w.id AND o.status = 'ACTIVE') AS active_orders,
          (SELECT coalesce(sum(octet_length(sp.photo_data)), 0)::bigint FROM stage_photos sp WHERE sp.workshop_id = w.id) AS photo_bytes
    ) metrics
    WHERE coalesce(search_text, '') = '' OR w.name ILIKE '%' || search_text || '%' OR owner.email::text ILIKE '%' || search_text || '%'
    ORDER BY w.created_at DESC
    LIMIT least(greatest(row_limit, 1), 100) OFFSET greatest(row_offset, 0)
$$;

CREATE OR REPLACE FUNCTION console_set_workshop_status(target_id uuid, new_status text, change_reason text, actor_id uuid)
RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp AS $$
BEGIN
    IF new_status NOT IN ('ACTIVE', 'SUSPENDED', 'PENDING_DELETION') THEN RAISE EXCEPTION 'invalid workshop status'; END IF;
    IF length(btrim(change_reason)) < 5 THEN RAISE EXCEPTION 'reason is required'; END IF;
    UPDATE workshops
       SET status = new_status, status_reason = btrim(change_reason), status_changed_at = now(),
           deletion_scheduled_for = CASE WHEN new_status = 'PENDING_DELETION' THEN now() + interval '30 days' ELSE NULL END,
           updated_at = now()
     WHERE id = target_id;
    IF NOT FOUND THEN RAISE EXCEPTION 'workshop not found'; END IF;
    IF new_status <> 'ACTIVE' THEN
        UPDATE user_sessions SET revoked_at = now()
         WHERE revoked_at IS NULL AND user_id IN (SELECT user_id FROM workshop_members WHERE workshop_id = target_id);
    END IF;
    INSERT INTO console_audit_events(actor_admin_id, workshop_id, event_type, reason)
    VALUES (actor_id, target_id, 'WORKSHOP_STATUS_CHANGED', btrim(change_reason));
END $$;

CREATE OR REPLACE FUNCTION console_purge_workshop(target_id uuid, confirm_name text, actor_id uuid)
RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp AS $$
DECLARE workshop_name text; scheduled timestamptz;
BEGIN
    SELECT name, deletion_scheduled_for INTO workshop_name, scheduled FROM workshops WHERE id = target_id FOR UPDATE;
    IF workshop_name IS NULL OR workshop_name <> confirm_name THEN RAISE EXCEPTION 'workshop confirmation does not match'; END IF;
    IF scheduled IS NULL OR scheduled > now() THEN RAISE EXCEPTION 'deletion waiting period has not elapsed'; END IF;
    INSERT INTO console_audit_events(actor_admin_id, workshop_id, event_type, reason, details)
    VALUES (actor_id, target_id, 'WORKSHOP_PURGED', 'Permanent deletion after waiting period', jsonb_build_object('name', workshop_name));
    DELETE FROM audit_events WHERE workshop_id = target_id;
    DELETE FROM workshops WHERE id = target_id;
    DELETE FROM users u WHERE NOT EXISTS (SELECT 1 FROM workshop_members wm WHERE wm.user_id = u.id);
END $$;

REVOKE ALL ON FUNCTION console_list_workshops(text, integer, integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION console_set_workshop_status(uuid, text, text, uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION console_purge_workshop(uuid, text, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION console_list_workshops(text, integer, integer) TO tallerflow_console;
GRANT EXECUTE ON FUNCTION console_set_workshop_status(uuid, text, text, uuid) TO tallerflow_console;
GRANT EXECUTE ON FUNCTION console_purge_workshop(uuid, text, uuid) TO tallerflow_console;
