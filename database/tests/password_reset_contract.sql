\set ON_ERROR_STOP on
BEGIN;

DO $$
BEGIN
    -- 1. Ensure password_reset_tokens table exists
    IF to_regclass('public.password_reset_tokens') IS NULL THEN
        RAISE EXCEPTION 'missing password_reset_tokens table';
    END IF;

    -- 2. Verify expected columns exist with correct types
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'password_reset_tokens' AND column_name = 'id'
    ) THEN
        RAISE EXCEPTION 'password_reset_tokens.id is required';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'password_reset_tokens' AND column_name = 'user_id'
    ) THEN
        RAISE EXCEPTION 'password_reset_tokens.user_id is required';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'password_reset_tokens' AND column_name = 'token_hash'
          AND data_type = 'bytea'
    ) THEN
        RAISE EXCEPTION 'password_reset_tokens.token_hash must be bytea';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'password_reset_tokens' AND column_name = 'expires_at'
    ) THEN
        RAISE EXCEPTION 'password_reset_tokens.expires_at is required';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'password_reset_tokens' AND column_name = 'used_at'
    ) THEN
        RAISE EXCEPTION 'password_reset_tokens.used_at is required';
    END IF;

    -- 3. Verify tallerflow_app permissions (SELECT, INSERT, UPDATE, DELETE)
    IF NOT has_table_privilege('tallerflow_app', 'public.password_reset_tokens', 'SELECT')
       OR NOT has_table_privilege('tallerflow_app', 'public.password_reset_tokens', 'INSERT')
       OR NOT has_table_privilege('tallerflow_app', 'public.password_reset_tokens', 'UPDATE')
       OR NOT has_table_privilege('tallerflow_app', 'public.password_reset_tokens', 'DELETE') THEN
        RAISE EXCEPTION 'tallerflow_app lacks required permissions on password_reset_tokens';
    END IF;

    -- 4. Verify user_id index exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public' AND tablename = 'password_reset_tokens'
          AND indexname = 'password_reset_tokens_user_id_idx'
    ) THEN
        RAISE EXCEPTION 'missing password_reset_tokens_user_id_idx index';
    END IF;
END $$;

-- 5. Verify check constraint: expires_at must be strictly greater than created_at
DO $$
DECLARE
    test_user_id uuid;
    sample_hash bytea := '\xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789';
    now_ts timestamptz := clock_timestamp();
BEGIN
    INSERT INTO public.users (email, name)
    VALUES ('token-contract-test@example.com', 'Token Contract User')
    RETURNING id INTO test_user_id;

    -- Insert valid token
    INSERT INTO public.password_reset_tokens (user_id, token_hash, expires_at, created_at)
    VALUES (test_user_id, sample_hash, now_ts + interval '1 hour', now_ts);

    -- Reject duplicate token_hash
    BEGIN
        INSERT INTO public.password_reset_tokens (user_id, token_hash, expires_at, created_at)
        VALUES (test_user_id, sample_hash, now_ts + interval '2 hours', now_ts);
        RAISE EXCEPTION 'duplicate token_hash was unexpectedly accepted';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;

    -- Reject expires_at <= created_at
    BEGIN
        INSERT INTO public.password_reset_tokens (user_id, token_hash, expires_at, created_at)
        VALUES (test_user_id, '\x1122334455667788990011223344556677889900112233445566778899001122', now_ts, now_ts);
        RAISE EXCEPTION 'expires_at equal to created_at was unexpectedly accepted';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END $$;

ROLLBACK;
