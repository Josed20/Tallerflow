CREATE TABLE analytics_play_daily (
 report_date date NOT NULL, package_name text NOT NULL CHECK(package_name='pe.tallerflow.app'),
 dimension text NOT NULL CHECK(dimension IN ('all','app_version')), version varchar(64) NOT NULL,
 device_installs bigint CHECK(device_installs>=0), user_installs bigint CHECK(user_installs>=0),
 uninstalls bigint CHECK(uninstalls>=0), source text NOT NULL DEFAULT 'google_play_csv' CHECK(source='google_play_csv'),
 source_name varchar(128) NOT NULL, imported_at timestamptz NOT NULL DEFAULT now(),
 imported_by uuid REFERENCES console_admins(id) ON DELETE SET NULL, checksum text NOT NULL,
 PRIMARY KEY(report_date,package_name,dimension,version)
);
REVOKE ALL ON analytics_play_daily FROM PUBLIC;
GRANT SELECT,INSERT,UPDATE ON analytics_play_daily TO tallerflow_console;

CREATE FUNCTION analytics_prune() RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=public,pg_temp AS $$
BEGIN
 DELETE FROM analytics_events WHERE occurred_at<now()-interval '90 days';
 DELETE FROM analytics_play_daily WHERE report_date<(now()-interval '25 months')::date;
END $$;
REVOKE ALL ON FUNCTION analytics_prune() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION analytics_prune() TO tallerflow_app;

CREATE FUNCTION console_analytics(start_day date,end_day date,client text,app_version text,include_demo boolean)
RETURNS TABLE(day date,first_opens bigint,registrations bigint,logins bigint,activities bigint,active_users bigint,active_workshops bigint,
 play_device_installs bigint,play_user_installs bigint,play_uninstalls bigint,play_imported_at timestamptz,coverage_started_at timestamptz)
LANGUAGE sql SECURITY DEFINER SET search_path=public,pg_temp AS $$
 WITH days AS (SELECT generate_series(start_day,end_day,interval '1 day')::date d),
 events AS (
  SELECT (e.occurred_at AT TIME ZONE 'UTC')::date d,
   count(*) FILTER(WHERE e.event_type='first_open') opens,count(*) FILTER(WHERE e.event_type='registration') registrations,
   count(*) FILTER(WHERE e.event_type='login') logins,count(*) FILTER(WHERE e.event_type='activity') activities,
   count(DISTINCT e.user_id) FILTER(WHERE e.event_type='activity') users,
   count(DISTINCT e.workshop_id) FILTER(WHERE e.event_type='activity') workshops
  FROM analytics_events e LEFT JOIN workshops w ON w.id=e.workshop_id
  WHERE e.occurred_at>=start_day::timestamp AT TIME ZONE 'UTC' AND e.occurred_at<(end_day+1)::timestamp AT TIME ZONE 'UTC'
   AND (client='' OR e.channel=client) AND (app_version='' OR e.version=app_version)
   AND (include_demo OR NOT coalesce(w.is_demo,false)) GROUP BY 1
 ), selected_play AS (
  SELECT p.* FROM analytics_play_daily p WHERE p.report_date BETWEEN start_day AND end_day
   AND client IN ('','android') AND (app_version='' OR (p.dimension='app_version' AND p.version=app_version))
   AND (app_version<>'' OR p.dimension='all' OR NOT EXISTS(SELECT 1 FROM analytics_play_daily t WHERE t.report_date=p.report_date AND t.dimension='all'))
 ), play AS (
  SELECT report_date d,
   CASE WHEN count(device_installs)=count(*) THEN sum(device_installs)::bigint END installs,
   CASE WHEN count(user_installs)=count(*) THEN sum(user_installs)::bigint END users,
   CASE WHEN count(uninstalls)=count(*) THEN sum(uninstalls)::bigint END uninstalls,max(imported_at) imported
  FROM selected_play GROUP BY report_date
 )
 SELECT days.d,
 CASE WHEN days.d>=greatest((c.started_at AT TIME ZONE 'UTC')::date,(now() AT TIME ZONE 'UTC')::date-89) THEN coalesce(e.opens,0) END,
 CASE WHEN days.d>=greatest((c.started_at AT TIME ZONE 'UTC')::date,(now() AT TIME ZONE 'UTC')::date-89) THEN coalesce(e.registrations,0) END,
 CASE WHEN days.d>=greatest((c.started_at AT TIME ZONE 'UTC')::date,(now() AT TIME ZONE 'UTC')::date-89) THEN coalesce(e.logins,0) END,
 CASE WHEN days.d>=greatest((c.started_at AT TIME ZONE 'UTC')::date,(now() AT TIME ZONE 'UTC')::date-89) THEN coalesce(e.activities,0) END,
 CASE WHEN days.d>=greatest((c.started_at AT TIME ZONE 'UTC')::date,(now() AT TIME ZONE 'UTC')::date-89) THEN coalesce(e.users,0) END,
 CASE WHEN days.d>=greatest((c.started_at AT TIME ZONE 'UTC')::date,(now() AT TIME ZONE 'UTC')::date-89) THEN coalesce(e.workshops,0) END,
 p.installs,p.users,p.uninstalls,p.imported,c.started_at
 FROM days CROSS JOIN analytics_coverage c LEFT JOIN events e ON e.d=days.d LEFT JOIN play p ON p.d=days.d ORDER BY days.d
$$;
REVOKE ALL ON FUNCTION console_analytics(date,date,text,text,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION console_analytics(date,date,text,text,boolean) TO tallerflow_console;

CREATE FUNCTION console_analytics_totals(start_day date,end_day date,client text,app_version text,include_demo boolean)
RETURNS TABLE(active_users bigint,active_workshops bigint,coverage_complete boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=public,pg_temp AS $$
 WITH coverage AS (SELECT start_day >= greatest((started_at AT TIME ZONE 'UTC')::date,(now() AT TIME ZONE 'UTC')::date-89) complete FROM analytics_coverage),
 counts AS (SELECT count(DISTINCT e.user_id) users,count(DISTINCT e.workshop_id) workshops
 FROM analytics_events e JOIN workshops w ON w.id=e.workshop_id
 WHERE e.event_type='activity' AND e.occurred_at>=start_day::timestamp AT TIME ZONE 'UTC'
 AND e.occurred_at<(end_day+1)::timestamp AT TIME ZONE 'UTC' AND (client='' OR e.channel=client)
 AND (app_version='' OR e.version=app_version) AND (include_demo OR NOT w.is_demo))
 SELECT CASE WHEN c.complete THEN n.users END, CASE WHEN c.complete THEN n.workshops END,c.complete FROM coverage c CROSS JOIN counts n
$$;
REVOKE ALL ON FUNCTION console_analytics_totals(date,date,text,text,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION console_analytics_totals(date,date,text,text,boolean) TO tallerflow_console;
