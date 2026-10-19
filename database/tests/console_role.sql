DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_roles
        WHERE rolname = 'tallerflow_console'
          AND rolcanlogin
          AND NOT rolsuper
          AND NOT rolbypassrls
          AND NOT rolinherit
    ) THEN
        RAISE EXCEPTION 'Console role must be a restricted login';
    END IF;

    IF NOT has_database_privilege('tallerflow_console', current_database(), 'CONNECT') THEN
        RAISE EXCEPTION 'Console role cannot connect';
    END IF;

    IF has_table_privilege('tallerflow_console', 'public.orders', 'SELECT')
       OR has_table_privilege('tallerflow_console', 'public.clients', 'SELECT')
       OR has_table_privilege('tallerflow_console', 'public.stage_photos', 'SELECT') THEN
        RAISE EXCEPTION 'Console role can read private workshop data';
    END IF;
END $$;
