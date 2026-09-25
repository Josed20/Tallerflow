\set ON_ERROR_STOP on
BEGIN;

DO $$
DECLARE
    required_table text;
BEGIN
    FOREACH required_table IN ARRAY ARRAY[
        'users',
        'user_credentials',
        'user_sessions',
        'workshops',
        'workshop_members',
        'audit_events'
    ]
    LOOP
        IF to_regclass('public.' || required_table) IS NULL THEN
            RAISE EXCEPTION 'missing identity contract table: %', required_table;
        END IF;
    END LOOP;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'name'
    ) THEN
        RAISE EXCEPTION 'users.name is required by the auth contract';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'workshops' AND column_name = 'timezone'
    ) THEN
        RAISE EXCEPTION 'workshops.timezone is required by the auth contract';
    END IF;

    IF NOT has_table_privilege('tallerflow_bootstrap', 'public.users', 'INSERT')
       OR NOT has_table_privilege('tallerflow_bootstrap', 'public.user_credentials', 'INSERT')
       OR NOT has_table_privilege('tallerflow_bootstrap', 'public.workshops', 'INSERT')
       OR NOT has_table_privilege('tallerflow_bootstrap', 'public.workshop_members', 'INSERT')
       OR NOT has_table_privilege('tallerflow_bootstrap', 'public.audit_events', 'INSERT') THEN
        RAISE EXCEPTION 'bootstrap role lacks required insert privileges';
    END IF;
END $$;

ROLLBACK;
