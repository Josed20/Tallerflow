\set ON_ERROR_STOP on
BEGIN;

DO $$
DECLARE
    test_user_id uuid;
    sample_hash bytea := '\x9988776655443322110099887766554433221100998877665544332211009988';
    token_row record;
BEGIN
    -- Ensure tenant context is empty
    PERFORM set_config('app.workshop_id', '', true);

    -- An application user creates a user
    INSERT INTO public.users (email, name)
    VALUES ('isolation-test@example.com', 'Isolation Test User')
    RETURNING id INTO test_user_id;

    -- Application user inserts a reset token without tenant context
    INSERT INTO public.password_reset_tokens (user_id, token_hash, expires_at)
    VALUES (test_user_id, sample_hash, now() + interval '1 hour')
    RETURNING * INTO token_row;

    IF token_row.id IS NULL THEN
        RAISE EXCEPTION 'failed to insert password_reset_token without tenant context';
    END IF;

    -- Lookup token without tenant context
    SELECT * INTO token_row
    FROM public.password_reset_tokens
    WHERE token_hash = sample_hash AND used_at IS NULL AND expires_at > now();

    IF token_row.id IS NULL THEN
        RAISE EXCEPTION 'failed to query password_reset_token without tenant context';
    END IF;

    -- Consume token without tenant context
    UPDATE public.password_reset_tokens
    SET used_at = now()
    WHERE id = token_row.id AND used_at IS NULL;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'failed to update used_at for reset token';
    END IF;

    -- Token must now be marked as used
    IF NOT EXISTS (
        SELECT 1 FROM public.password_reset_tokens
        WHERE id = token_row.id AND used_at IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'token was not marked as used';
    END IF;
END $$;

ROLLBACK;
