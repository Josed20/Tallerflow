ALTER TABLE console_audit_events ADD COLUMN workshop_name text;

CREATE OR REPLACE FUNCTION console_summary()
RETURNS TABLE(active_workshops bigint, suspended_workshops bigint, pending_deletion_workshops bigint,
              created_last_7_days bigint, created_last_30_days bigint, active_last_7_days bigint,
              unverified_owners_over_48h bigint, orders_last_30_days bigint)
LANGUAGE sql SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT
    count(*) FILTER (WHERE w.status = 'ACTIVE'),
    count(*) FILTER (WHERE w.status = 'SUSPENDED'),
    count(*) FILTER (WHERE w.status = 'PENDING_DELETION'),
    count(*) FILTER (WHERE w.created_at >= now() - interval '7 days'),
    count(*) FILTER (WHERE w.created_at >= now() - interval '30 days'),
    count(*) FILTER (WHERE activity.last_activity >= now() - interval '7 days'),
    (SELECT count(*) FROM workshops uw JOIN workshop_members wm ON wm.workshop_id = uw.id AND wm.role = 'OWNER' AND wm.status = 'ACTIVE'
      JOIN users u ON u.id = wm.user_id WHERE u.email_verified_at IS NULL AND u.created_at < now() - interval '48 hours'),
    (SELECT count(*) FROM orders o WHERE o.created_at >= now() - interval '30 days')
  FROM workshops w
  LEFT JOIN LATERAL (SELECT greatest(max(us.last_seen_at), max(o.updated_at)) AS last_activity
    FROM workshop_members wm LEFT JOIN user_sessions us ON us.user_id = wm.user_id
    LEFT JOIN orders o ON o.workshop_id = w.id WHERE wm.workshop_id = w.id) activity ON true
$$;

CREATE OR REPLACE FUNCTION console_workshop_detail(target_id uuid)
RETURNS TABLE(id uuid, name text, owner_email citext, created_at timestamptz, last_activity timestamptz,
              members bigint, active_orders bigint, photo_bytes bigint, status text,
              deletion_scheduled_for timestamptz, status_reason text, status_changed_at timestamptz)
LANGUAGE sql SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT w.id, w.name, owner.email, w.created_at, greatest(activity.last_session, activity.last_order),
    metrics.members, metrics.active_orders, metrics.photo_bytes, w.status, w.deletion_scheduled_for,
    w.status_reason, w.status_changed_at
  FROM workshops w
  LEFT JOIN LATERAL (SELECT u.email FROM workshop_members wm JOIN users u ON u.id=wm.user_id
    WHERE wm.workshop_id=w.id AND wm.role='OWNER' AND wm.status='ACTIVE' ORDER BY wm.created_at LIMIT 1) owner ON true
  LEFT JOIN LATERAL (SELECT max(us.last_seen_at) last_session, max(o.updated_at) last_order
    FROM workshop_members wm LEFT JOIN user_sessions us ON us.user_id=wm.user_id
    LEFT JOIN orders o ON o.workshop_id=w.id WHERE wm.workshop_id=w.id) activity ON true
  CROSS JOIN LATERAL (SELECT (SELECT count(*) FROM workshop_members wm WHERE wm.workshop_id=w.id AND wm.status='ACTIVE') members,
    (SELECT count(*) FROM orders o WHERE o.workshop_id=w.id AND o.status='ACTIVE') active_orders,
    (SELECT coalesce(sum(octet_length(sp.photo_data)),0)::bigint FROM stage_photos sp WHERE sp.workshop_id=w.id) photo_bytes) metrics
  WHERE w.id=target_id
$$;

CREATE OR REPLACE FUNCTION console_set_workshop_status(target_id uuid, new_status text, change_reason text, actor_id uuid, source_ip text)
RETURNS TABLE(owner_email citext, workshop_name text, deletion_scheduled_for timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp AS $$
DECLARE old_status text;
BEGIN
  IF new_status NOT IN ('ACTIVE','SUSPENDED','PENDING_DELETION') THEN RAISE EXCEPTION 'invalid workshop status'; END IF;
  IF length(btrim(change_reason)) < 5 THEN RAISE EXCEPTION 'reason is required'; END IF;
  SELECT w.status, w.name INTO old_status, workshop_name FROM workshops w WHERE w.id=target_id FOR UPDATE;
  IF workshop_name IS NULL THEN RAISE EXCEPTION 'workshop not found'; END IF;
  UPDATE workshops w SET status=new_status, status_reason=btrim(change_reason), status_changed_at=now(),
    deletion_scheduled_for=CASE WHEN new_status='PENDING_DELETION' THEN now()+interval '30 days' ELSE NULL END, updated_at=now()
    WHERE w.id=target_id RETURNING w.deletion_scheduled_for INTO deletion_scheduled_for;
  SELECT u.email INTO owner_email FROM workshop_members wm JOIN users u ON u.id=wm.user_id
    WHERE wm.workshop_id=target_id AND wm.role='OWNER' AND wm.status='ACTIVE' ORDER BY wm.created_at LIMIT 1;
  IF new_status <> 'ACTIVE' THEN UPDATE user_sessions SET revoked_at=now() WHERE revoked_at IS NULL
    AND user_id IN (SELECT user_id FROM workshop_members WHERE workshop_id=target_id); END IF;
  INSERT INTO console_audit_events(actor_admin_id, workshop_id, workshop_name, event_type, reason, source_ip, details)
  VALUES(actor_id,target_id,workshop_name,
    CASE WHEN old_status='PENDING_DELETION' AND new_status='ACTIVE' THEN 'WORKSHOP_DELETION_CANCELLED' ELSE 'WORKSHOP_STATUS_CHANGED' END,
    btrim(change_reason),source_ip,jsonb_build_object('previous_status',old_status,'new_status',new_status));
  RETURN NEXT;
END $$;

CREATE OR REPLACE FUNCTION console_purge_workshop(target_id uuid, confirm_name text, actor_id uuid, source_ip text)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp AS $$
DECLARE workshop_name text; scheduled timestamptz;
BEGIN
  SELECT name,deletion_scheduled_for INTO workshop_name,scheduled FROM workshops WHERE id=target_id FOR UPDATE;
  IF workshop_name IS NULL OR workshop_name<>confirm_name THEN RAISE EXCEPTION 'workshop confirmation does not match'; END IF;
  IF scheduled IS NULL OR scheduled>now() THEN RAISE EXCEPTION 'deletion waiting period has not elapsed'; END IF;
  INSERT INTO console_audit_events(actor_admin_id,workshop_id,workshop_name,event_type,reason,source_ip,details)
    VALUES(actor_id,target_id,workshop_name,'WORKSHOP_PURGED','Permanent deletion after waiting period',source_ip,jsonb_build_object('name',workshop_name));
  DELETE FROM audit_events WHERE workshop_id=target_id;
  DELETE FROM workshops WHERE id=target_id;
  DELETE FROM users u WHERE NOT EXISTS(SELECT 1 FROM workshop_members wm WHERE wm.user_id=u.id);
END $$;

REVOKE ALL ON FUNCTION console_summary() FROM PUBLIC;
REVOKE ALL ON FUNCTION console_workshop_detail(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION console_set_workshop_status(uuid,text,text,uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION console_set_workshop_status(uuid,text,text,uuid,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION console_purge_workshop(uuid,text,uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION console_purge_workshop(uuid,text,uuid,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION console_summary() TO tallerflow_console;
GRANT EXECUTE ON FUNCTION console_workshop_detail(uuid) TO tallerflow_console;
GRANT EXECUTE ON FUNCTION console_set_workshop_status(uuid,text,text,uuid,text) TO tallerflow_console;
GRANT EXECUTE ON FUNCTION console_purge_workshop(uuid,text,uuid,text) TO tallerflow_console;
