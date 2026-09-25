\set ON_ERROR_STOP on
BEGIN;

DO $$
BEGIN
    IF (SELECT count(*) FROM public.workshops) <> 0 THEN
        RAISE EXCEPTION 'RLS exposed rows without tenant context';
    END IF;
END $$;

SELECT set_config('app.workshop_id', '10000000-0000-0000-0000-000000000001', true);

DO $$
BEGIN
    IF (SELECT count(*) FROM public.workshops) <> 1 THEN
        RAISE EXCEPTION 'RLS did not isolate the active workshop';
    END IF;
    IF EXISTS (
        SELECT 1 FROM public.workshops
        WHERE id = '20000000-0000-0000-0000-000000000002'
    ) THEN
        RAISE EXCEPTION 'RLS exposed a different workshop';
    END IF;
END $$;

ROLLBACK;
