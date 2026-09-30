\set ON_ERROR_STOP on
BEGIN;

DO $$
DECLARE
    test_user uuid;
BEGIN
    IF to_regclass('public.password_reset_tokens') IS NULL THEN
        RAISE EXCEPTION 'missing password_reset_tokens';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND tablename='password_reset_tokens' AND indexname='password_reset_tokens_user_id_idx') THEN
        RAISE EXCEPTION 'missing password reset user index';
    END IF;
    IF NOT has_table_privilege('tallerflow_app', 'public.password_reset_tokens', 'SELECT,INSERT,UPDATE') THEN
        RAISE EXCEPTION 'application role lacks password reset privileges';
    END IF;

    INSERT INTO users(email,name) VALUES ('reset-contract@example.test','Reset Contract') RETURNING id INTO test_user;
    INSERT INTO password_reset_tokens(user_id,token_hash,expires_at) VALUES (test_user,digest('opaque-test-token','sha256'),now()+interval '1 hour');
    IF EXISTS (SELECT 1 FROM password_reset_tokens WHERE token_hash=convert_to('opaque-test-token','UTF8')) THEN
        RAISE EXCEPTION 'raw reset token was stored';
    END IF;
    BEGIN
        INSERT INTO password_reset_tokens(user_id,token_hash,expires_at) VALUES (test_user,digest('expired','sha256'),now()-interval '1 minute');
        RAISE EXCEPTION 'invalid expiry was accepted';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
END $$;

ROLLBACK;
