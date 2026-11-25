-- Raw telemetry is deliberately separate from operational data and Play reports.
ALTER TABLE workshops ADD COLUMN is_demo boolean NOT NULL DEFAULT false;
CREATE TABLE analytics_events (
 id uuid PRIMARY KEY, installation_id uuid NOT NULL,
 event_type text NOT NULL CHECK (event_type IN ('first_open','registration','login','activity')),
 occurred_at timestamptz NOT NULL, received_at timestamptz NOT NULL DEFAULT now(),
 channel text NOT NULL CHECK (channel IN ('web','android')), version varchar(64) NOT NULL,
 workshop_id uuid REFERENCES workshops(id) ON DELETE CASCADE,
 user_id uuid REFERENCES users(id) ON DELETE CASCADE,
 source_event_id uuid,
 CHECK ((event_type='first_open' AND workshop_id IS NULL AND user_id IS NULL) OR
        (event_type<>'first_open' AND workshop_id IS NOT NULL AND user_id IS NOT NULL)),
 CHECK (occurred_at >= received_at - interval '7 days' AND occurred_at <= received_at + interval '5 minutes')
);
CREATE UNIQUE INDEX analytics_first_open_installation ON analytics_events(installation_id) WHERE event_type='first_open';
CREATE UNIQUE INDEX analytics_login_evidence ON analytics_events(source_event_id) WHERE event_type='login';
CREATE UNIQUE INDEX analytics_registration_workshop ON analytics_events(workshop_id) WHERE event_type='registration';
CREATE INDEX analytics_events_period ON analytics_events(occurred_at,channel,version);
CREATE INDEX analytics_events_tenant ON analytics_events(workshop_id,occurred_at);
ALTER TABLE analytics_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_events FORCE ROW LEVEL SECURITY;
CREATE POLICY analytics_events_tenant ON analytics_events FOR ALL TO tallerflow_app
 USING (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid)
 WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid
 AND EXISTS (SELECT 1 FROM workshop_members wm WHERE wm.workshop_id=analytics_events.workshop_id AND wm.user_id=analytics_events.user_id AND wm.status='ACTIVE'));
CREATE POLICY analytics_events_metrics ON analytics_events TO tallerflow_migrator USING (true) WITH CHECK (true);
-- Tenant writes only through the evidence-validating function below.
GRANT SELECT ON analytics_events TO tallerflow_app;
CREATE TABLE analytics_coverage (started_at timestamptz NOT NULL DEFAULT now());
INSERT INTO analytics_coverage DEFAULT VALUES;
REVOKE ALL ON analytics_events, analytics_coverage FROM PUBLIC;

CREATE FUNCTION analytics_first_open(event_id uuid, installation uuid, occurred timestamptz, client text, app_version text)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=public,pg_temp AS $$
BEGIN
 IF client NOT IN ('web','android') OR app_version !~ '^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$'
 OR occurred IS NULL OR occurred < now()-interval '7 days' OR occurred > now()+interval '5 minutes' THEN RAISE EXCEPTION 'invalid event'; END IF;
 INSERT INTO analytics_events(id,installation_id,event_type,occurred_at,channel,version)
 VALUES(event_id,installation,'first_open',occurred,client,app_version) ON CONFLICT DO NOTHING;
END $$;
REVOKE ALL ON FUNCTION analytics_first_open(uuid,uuid,timestamptz,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION analytics_first_open(uuid,uuid,timestamptz,text,text) TO tallerflow_app;

CREATE FUNCTION console_summary_filtered(include_demo boolean)
RETURNS TABLE(active_workshops bigint,suspended_workshops bigint,pending_deletion_workshops bigint,
 created_last_7_days bigint,created_last_30_days bigint,active_last_7_days bigint,unverified_owners_over_48h bigint,orders_last_30_days bigint)
LANGUAGE sql SECURITY DEFINER SET search_path=public,pg_temp AS $$
 SELECT count(*) FILTER(WHERE w.status='ACTIVE'), count(*) FILTER(WHERE w.status='SUSPENDED'),count(*) FILTER(WHERE w.status='PENDING_DELETION'),
 count(*) FILTER(WHERE w.created_at>=now()-interval '7 days'),count(*) FILTER(WHERE w.created_at>=now()-interval '30 days'),
 count(*) FILTER(WHERE EXISTS(SELECT 1 FROM analytics_events e WHERE e.workshop_id=w.id AND e.event_type='activity' AND e.occurred_at>=now()-interval '7 days')
 OR EXISTS(SELECT 1 FROM orders o WHERE o.workshop_id=w.id AND o.updated_at>=now()-interval '7 days')),
 (SELECT count(DISTINCT uw.id) FROM workshops uw JOIN workshop_members wm ON wm.workshop_id=uw.id AND wm.role='OWNER' AND wm.status='ACTIVE'
 JOIN users u ON u.id=wm.user_id WHERE u.email_verified_at IS NULL AND u.created_at<now()-interval '48 hours' AND (include_demo OR NOT uw.is_demo)),
 (SELECT count(*) FROM orders o JOIN workshops ow ON ow.id=o.workshop_id WHERE o.created_at>=now()-interval '30 days' AND (include_demo OR NOT ow.is_demo))
 FROM workshops w WHERE include_demo OR NOT w.is_demo
$$;
CREATE OR REPLACE FUNCTION console_summary() RETURNS TABLE(active_workshops bigint,suspended_workshops bigint,pending_deletion_workshops bigint,
 created_last_7_days bigint,created_last_30_days bigint,active_last_7_days bigint,unverified_owners_over_48h bigint,orders_last_30_days bigint)
 LANGUAGE sql SECURITY DEFINER SET search_path=public,pg_temp AS $$ SELECT * FROM console_summary_filtered(false) $$;
REVOKE ALL ON FUNCTION console_summary_filtered(boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION console_summary_filtered(boolean) TO tallerflow_console;

CREATE FUNCTION console_set_demo(target uuid, demo boolean, actor uuid)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=public,pg_temp AS $$
BEGIN
 UPDATE workshops SET is_demo=demo WHERE id=target;
 IF NOT FOUND THEN RAISE EXCEPTION 'workshop not found'; END IF;
 INSERT INTO console_audit_events(actor_admin_id,workshop_id,event_type,details)
 VALUES(actor,target,'WORKSHOP_DEMO_CHANGED',jsonb_build_object('is_demo',demo));
END $$;
REVOKE ALL ON FUNCTION console_set_demo(uuid,boolean,uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION console_set_demo(uuid,boolean,uuid) TO tallerflow_console;

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
        SELECT (SELECT max(e.occurred_at) FROM analytics_events e WHERE e.workshop_id=w.id AND e.event_type='activity') AS last_session, max(o.updated_at) AS last_order
        FROM workshop_members wm
        LEFT JOIN user_sessions us ON us.user_id = wm.user_id
        LEFT JOIN orders o ON o.workshop_id = w.id
        WHERE wm.workshop_id = w.id
    ) activity ON true
    CROSS JOIN LATERAL (
        SELECT
          (SELECT count(*) FROM workshop_members wm WHERE wm.workshop_id = w.id AND wm.status = 'ACTIVE') AS members,
          (SELECT count(*) FROM orders o WHERE o.workshop_id = w.id AND o.status = 'ACTIVE') AS active_orders,
          (SELECT CASE WHEN count(*) FILTER (WHERE sp.object_key IS NOT NULL)>0 THEN NULL ELSE coalesce(sum(octet_length(sp.photo_data)),0)::bigint END FROM stage_photos sp WHERE sp.workshop_id = w.id) AS photo_bytes
    ) metrics
    WHERE coalesce(search_text, '') = '' OR w.name ILIKE '%' || search_text || '%' OR owner.email::text ILIKE '%' || search_text || '%'
    ORDER BY w.created_at DESC
    LIMIT least(greatest(row_limit, 1), 100) OFFSET greatest(row_offset, 0)
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
  LEFT JOIN LATERAL (SELECT (SELECT max(e.occurred_at) FROM analytics_events e WHERE e.workshop_id=w.id AND e.event_type='activity') last_session, max(o.updated_at) last_order
    FROM workshop_members wm LEFT JOIN user_sessions us ON us.user_id=wm.user_id
    LEFT JOIN orders o ON o.workshop_id=w.id WHERE wm.workshop_id=w.id) activity ON true
  CROSS JOIN LATERAL (SELECT (SELECT count(*) FROM workshop_members wm WHERE wm.workshop_id=w.id AND wm.status='ACTIVE') members,
    (SELECT count(*) FROM orders o WHERE o.workshop_id=w.id AND o.status='ACTIVE') active_orders,
    (SELECT CASE WHEN count(*) FILTER (WHERE sp.object_key IS NOT NULL)>0 THEN NULL ELSE coalesce(sum(octet_length(sp.photo_data)),0)::bigint END FROM stage_photos sp WHERE sp.workshop_id=w.id) photo_bytes) metrics
  WHERE w.id=target_id
$$;

CREATE FUNCTION console_photo_counts(target uuid) RETURNS TABLE(photo_count bigint,r2_photo_count bigint,is_demo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=public,pg_temp AS $$
 SELECT count(sp.id),count(sp.id) FILTER(WHERE sp.object_key IS NOT NULL), w.is_demo
 FROM workshops w LEFT JOIN stage_photos sp ON sp.workshop_id=w.id WHERE w.id=target GROUP BY w.id,w.is_demo
$$;
REVOKE ALL ON FUNCTION console_photo_counts(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION console_photo_counts(uuid) TO tallerflow_console;

CREATE POLICY audit_analytics_evidence ON audit_events FOR SELECT TO tallerflow_migrator USING (true);
CREATE FUNCTION analytics_record_event(event_id uuid, installation uuid, kind text, occurred timestamptz, client text, app_version text, tenant uuid, actor uuid)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=public,pg_temp AS $$
DECLARE evidence uuid; actual_time timestamptz;
BEGIN
 IF tenant IS DISTINCT FROM nullif(current_setting('app.workshop_id',true),'')::uuid
 OR NOT EXISTS(SELECT 1 FROM workshop_members wm WHERE wm.workshop_id=tenant AND wm.user_id=actor AND wm.status='ACTIVE')
 OR kind NOT IN ('registration','login','activity') OR client NOT IN ('web','android')
 OR app_version !~ '^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$' OR occurred IS NULL
 OR occurred<now()-interval '7 days' OR occurred>now()+interval '5 minutes' THEN RETURN false; END IF;
 IF EXISTS(SELECT 1 FROM analytics_events e WHERE e.id=event_id AND e.workshop_id=tenant AND e.user_id=actor) THEN RETURN true; END IF;
 actual_time:=occurred;
 IF kind='login' THEN
  PERFORM pg_advisory_xact_lock(hashtextextended(tenant::text || ':' || actor::text,0));
  IF EXISTS(SELECT 1 FROM analytics_events e WHERE e.id=event_id AND e.workshop_id=tenant AND e.user_id=actor) THEN RETURN true; END IF;
  SELECT a.id,a.created_at INTO evidence,actual_time FROM audit_events a
  WHERE a.workshop_id=tenant AND a.actor_user_id=actor AND a.event_type='USER_SIGNED_IN'
   AND a.created_at BETWEEN occurred-interval '5 minutes' AND occurred+interval '5 minutes'
   AND a.details->>'client'=CASE WHEN client='android' THEN 'mobile' ELSE 'web' END
   AND NOT EXISTS (SELECT 1 FROM analytics_events consumed WHERE consumed.event_type='login' AND consumed.source_event_id=a.id)
  ORDER BY abs(extract(epoch FROM (a.created_at-occurred))) LIMIT 1;
  IF evidence IS NULL THEN RETURN false; END IF;
 ELSIF kind='registration' THEN
  SELECT u.email_verified_at INTO actual_time FROM users u JOIN workshop_members wm ON wm.user_id=u.id
  WHERE u.id=actor AND wm.workshop_id=tenant AND wm.role='OWNER' AND wm.status='ACTIVE'
   AND u.email_verified_at BETWEEN occurred-interval '5 minutes' AND occurred+interval '5 minutes';
  IF actual_time IS NULL THEN RETURN false; END IF;
 END IF;
 INSERT INTO analytics_events(id,installation_id,event_type,occurred_at,channel,version,workshop_id,user_id,source_event_id)
 VALUES(event_id,installation,kind,actual_time,client,app_version,tenant,actor,evidence) ON CONFLICT DO NOTHING;
 RETURN true;
END $$;
REVOKE ALL ON FUNCTION analytics_record_event(uuid,uuid,text,timestamptz,text,text,uuid,uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION analytics_record_event(uuid,uuid,text,timestamptz,text,text,uuid,uuid) TO tallerflow_app;
