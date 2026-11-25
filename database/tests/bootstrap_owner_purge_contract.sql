-- Run after V22 with psql -v ON_ERROR_STOP=1 as the isolated verification DB
-- superuser. Run separately from integration tests: the real console purge
-- visits orphan users. All fixtures and any existing bootstrap marker changes
-- are rolled back; no schema changes or persistent deletions are performed.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path TO public;

SELECT gen_random_uuid() AS purge_workshop, gen_random_uuid() AS keep_workshop,
       gen_random_uuid() AS first_owner, gen_random_uuid() AS keep_owner,
       gen_random_uuid() AS keep_client, gen_random_uuid() AS keep_order,
       gen_random_uuid() AS console_admin \gset
SELECT set_config('s22.purge_workshop', :'purge_workshop', true),
       set_config('s22.keep_workshop', :'keep_workshop', true),
       set_config('s22.first_owner', :'first_owner', true),
       set_config('s22.keep_owner', :'keep_owner', true),
       set_config('s22.keep_order', :'keep_order', true);

INSERT INTO users (id, email, name) VALUES
    (:'first_owner', :'first_owner' || '@s22-contract.test', 'Initial owner'),
    (:'keep_owner', :'keep_owner' || '@s22-contract.test', 'Other workshop owner');
INSERT INTO user_credentials (user_id, password_hash) VALUES
    (:'first_owner', 'contract-only'), (:'keep_owner', 'contract-only');
INSERT INTO workshops (id, name, status, deletion_scheduled_for) VALUES
    (:'purge_workshop', 's22-purge-' || :'purge_workshop', 'PENDING_DELETION', now() - interval '1 day'),
    (:'keep_workshop', 's22-keep-' || :'keep_workshop', 'ACTIVE', NULL);
INSERT INTO workshop_members (workshop_id, user_id, role) VALUES
    (:'purge_workshop', :'first_owner', 'OWNER'),
    (:'keep_workshop', :'keep_owner', 'OWNER');
INSERT INTO clients (id, workshop_id, name)
    VALUES (:'keep_client', :'keep_workshop', 'Surviving client');
INSERT INTO orders (id, workshop_id, client_id, order_number, code, product, quantity, unit, promised_at)
    VALUES (:'keep_order', :'keep_workshop', :'keep_client', 1, 'S22-1', 'Surviving order', 600, 'UNITS', now());
INSERT INTO console_admins (id, email, password_hash, totp_secret_ciphertext)
    VALUES (:'console_admin', :'console_admin' || '@s22-contract.test', 'contract-only', decode('01', 'hex'));

-- Exercise the singleton whether the verification DB is already claimed or not.
-- Preserve the original claim timestamp when a marker already exists.
INSERT INTO bootstrap_state (singleton, owner_user_id) VALUES (true, :'first_owner')
    ON CONFLICT (singleton) DO UPDATE SET owner_user_id = EXCLUDED.owner_user_id;
SELECT set_config('s22.claimed_at', bootstrapped_at::text, true)
    FROM bootstrap_state WHERE singleton = true;

-- Use the production SECURITY DEFINER entry point as its restricted caller.
SET LOCAL ROLE tallerflow_console;
SELECT console_purge_workshop(
    :'purge_workshop'::uuid, 's22-purge-' || :'purge_workshop',
    :'console_admin'::uuid, '127.0.0.1');
RESET ROLE;

DO $$ BEGIN
    IF EXISTS (SELECT FROM users WHERE id = current_setting('s22.first_owner')::uuid)
       OR EXISTS (SELECT FROM user_credentials WHERE user_id = current_setting('s22.first_owner')::uuid)
       OR EXISTS (SELECT FROM workshop_members WHERE workshop_id = current_setting('s22.purge_workshop')::uuid)
       OR EXISTS (SELECT FROM workshops WHERE id = current_setting('s22.purge_workshop')::uuid) THEN
        RAISE EXCEPTION 'Console purge failed to delete the initial owner and workshop';
    END IF;
    IF NOT EXISTS (
        SELECT FROM bootstrap_state
        WHERE singleton = true AND owner_user_id IS NULL
          AND bootstrapped_at = current_setting('s22.claimed_at')::timestamptz
    ) THEN
        RAISE EXCEPTION 'Deleting the initial owner must preserve the singleton and original claim timestamp';
    END IF;
    IF NOT EXISTS (SELECT FROM workshops WHERE id = current_setting('s22.keep_workshop')::uuid AND status = 'ACTIVE')
       OR NOT EXISTS (SELECT FROM users WHERE id = current_setting('s22.keep_owner')::uuid)
       OR NOT EXISTS (SELECT FROM user_credentials WHERE user_id = current_setting('s22.keep_owner')::uuid)
       OR NOT EXISTS (SELECT FROM workshop_members WHERE workshop_id = current_setting('s22.keep_workshop')::uuid AND user_id = current_setting('s22.keep_owner')::uuid AND role = 'OWNER' AND status = 'ACTIVE')
       OR NOT EXISTS (SELECT FROM orders WHERE id = current_setting('s22.keep_order')::uuid AND workshop_id = current_setting('s22.keep_workshop')::uuid AND product = 'Surviving order' AND quantity = 600) THEN
        RAISE EXCEPTION 'Purging the first workshop changed the other workshop or its owner/order';
    END IF;
    IF NOT EXISTS (SELECT FROM console_audit_events WHERE workshop_id = current_setting('s22.purge_workshop')::uuid AND event_type = 'WORKSHOP_PURGED') THEN
        RAISE EXCEPTION 'Console purge lost its administrative audit record';
    END IF;
END $$;

-- Match PostgresStatusStore.Claimed exactly, under the actual bootstrap role.
SET LOCAL ROLE tallerflow_bootstrap;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM bootstrap_state WHERE singleton = true) THEN
        RAISE EXCEPTION 'Onboarding would incorrectly consider the installation unclaimed';
    END IF;
    BEGIN
        INSERT INTO bootstrap_state (owner_user_id)
            VALUES (current_setting('s22.keep_owner')::uuid);
        RAISE EXCEPTION 'Installation was publicly reclaimable after initial owner deletion';
    EXCEPTION WHEN unique_violation THEN
        NULL; -- The surviving singleton must reject a second bootstrap claim.
    END;
END $$;
RESET ROLE;
ROLLBACK;
