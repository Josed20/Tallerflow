\set ON_ERROR_STOP on
BEGIN;

INSERT INTO workshops (id, name, slug) VALUES
    ('10000000-0000-0000-0000-000000000001', 'Workshop One', 'rls-workshop-one'),
    ('20000000-0000-0000-0000-000000000002', 'Workshop Two', 'rls-workshop-two');

SET LOCAL ROLE tallerflow_app;

DO $$
BEGIN
    IF (SELECT count(*) FROM workshops) <> 0 THEN
        RAISE EXCEPTION 'RLS exposed rows without tenant context';
    END IF;
END $$;

SELECT set_config('app.workshop_id', '10000000-0000-0000-0000-000000000001', true);

DO $$
BEGIN
    IF (SELECT count(*) FROM workshops) <> 1 THEN
        RAISE EXCEPTION 'RLS did not isolate the active workshop';
    END IF;
    IF EXISTS (
        SELECT 1 FROM workshops
        WHERE id = '20000000-0000-0000-0000-000000000002'
    ) THEN
        RAISE EXCEPTION 'RLS exposed a different workshop';
    END IF;
END $$;

RESET ROLE;
ROLLBACK;
