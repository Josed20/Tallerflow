\set ON_ERROR_STOP on
-- Run as tallerflow_app after V11.
BEGIN;

DO $$
BEGIN
    IF to_regclass('public.email_verification_tokens') IS NULL THEN
        RAISE EXCEPTION 'missing email_verification_tokens table';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'email_verification_tokens' AND column_name = 'token_hash'
          AND data_type = 'bytea'
    ) THEN
        RAISE EXCEPTION 'email_verification_tokens.token_hash must be bytea';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'email_verified_at'
    ) OR NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'terms_accepted_at'
    ) THEN
        RAISE EXCEPTION 'users.email_verified_at and users.terms_accepted_at are required';
    END IF;

    IF NOT has_table_privilege('tallerflow_app', 'public.email_verification_tokens', 'SELECT')
       OR NOT has_table_privilege('tallerflow_app', 'public.email_verification_tokens', 'INSERT')
       OR NOT has_table_privilege('tallerflow_app', 'public.email_verification_tokens', 'UPDATE')
       OR NOT has_table_privilege('tallerflow_app', 'public.email_verification_tokens', 'DELETE') THEN
        RAISE EXCEPTION 'tallerflow_app lacks required permissions on email_verification_tokens';
    END IF;

    IF EXISTS (SELECT 1 FROM public.users WHERE email_verified_at IS NULL) THEN
        RAISE EXCEPTION 'accounts created before V11 must stay verified';
    END IF;
END $$;

DO $$
DECLARE
    test_user_id uuid;
    now_ts timestamptz := clock_timestamp();
BEGIN
    INSERT INTO public.users (email, name)
    VALUES ('verification-contract-test@example.com', 'Verification Contract User')
    RETURNING id INTO test_user_id;

    IF (SELECT email_verified_at FROM public.users WHERE id = test_user_id) IS NOT NULL THEN
        RAISE EXCEPTION 'new users must start unverified';
    END IF;

    INSERT INTO public.email_verification_tokens (user_id, token_hash, expires_at, created_at)
    VALUES (test_user_id, '\xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789', now_ts + interval '24 hours', now_ts);

    -- Only one pending token per user.
    BEGIN
        INSERT INTO public.email_verification_tokens (user_id, token_hash, expires_at, created_at)
        VALUES (test_user_id, '\x1122334455667788990011223344556677889900112233445566778899001122', now_ts + interval '24 hours', now_ts);
        RAISE EXCEPTION 'a second pending token was unexpectedly accepted';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;

    -- Once used, a new token can be issued.
    UPDATE public.email_verification_tokens SET used_at = now_ts WHERE user_id = test_user_id;
    INSERT INTO public.email_verification_tokens (user_id, token_hash, expires_at, created_at)
    VALUES (test_user_id, '\x1122334455667788990011223344556677889900112233445566778899001122', now_ts + interval '24 hours', now_ts);
    UPDATE public.email_verification_tokens SET used_at = now_ts WHERE user_id = test_user_id AND used_at IS NULL;

    BEGIN
        INSERT INTO public.email_verification_tokens (user_id, token_hash, expires_at, created_at)
        VALUES (test_user_id, '\x99887766554433221100998877665544332211009988776655443322110099aa', now_ts, now_ts);
        RAISE EXCEPTION 'expires_at equal to created_at was unexpectedly accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;

    BEGIN
        UPDATE public.users SET email_verified_at = created_at - interval '1 second' WHERE id = test_user_id;
        RAISE EXCEPTION 'email_verified_at before created_at was unexpectedly accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END $$;

ROLLBACK;
