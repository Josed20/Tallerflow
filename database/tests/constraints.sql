\set ON_ERROR_STOP on
BEGIN;

DO $$
DECLARE
    test_user uuid;
    test_workshop uuid;
BEGIN
    INSERT INTO users (email, password_hash, display_name)
    VALUES ('constraint@example.test', '$argon2id$test', 'Constraint Test')
    RETURNING id INTO test_user;

    INSERT INTO workshops (name, slug)
    VALUES ('Constraint Workshop', 'constraint-workshop')
    RETURNING id INTO test_workshop;

    BEGIN
        INSERT INTO memberships (workshop_id, user_id, role)
        VALUES (test_workshop, test_user, 'INVALID');
        RAISE EXCEPTION 'invalid membership role was accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;

    BEGIN
        INSERT INTO users (email, password_hash, display_name)
        VALUES ('CONSTRAINT@example.test', '$argon2id$test', 'Bad Email');
        RAISE EXCEPTION 'non-normalized email was accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END $$;

ROLLBACK;

