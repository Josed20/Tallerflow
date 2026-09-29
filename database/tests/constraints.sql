\set ON_ERROR_STOP on
BEGIN;

DO $$
DECLARE
    test_user uuid;
    test_workshop uuid;
BEGIN
    INSERT INTO public.users (email, name)
    VALUES ('constraint@example.test', 'Constraint Test')
    RETURNING id INTO test_user;

    INSERT INTO public.workshops (name, timezone)
    VALUES ('Constraint Workshop', 'America/Lima')
    RETURNING id INTO test_workshop;

    BEGIN
        INSERT INTO public.workshop_members (workshop_id, user_id, role)
        VALUES (test_workshop, test_user, 'INVALID');
        RAISE EXCEPTION 'invalid membership role was accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;

    BEGIN
        INSERT INTO public.users (email, name)
        VALUES ('CONSTRAINT@example.test', 'Duplicate Email');
        RAISE EXCEPTION 'case-insensitive duplicate email was accepted';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;
END $$;

ROLLBACK;
