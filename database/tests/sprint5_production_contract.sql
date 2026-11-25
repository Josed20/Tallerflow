-- Run with psql -v ON_ERROR_STOP=1 as the isolated verification DB superuser.
-- No persistent fixtures or schema changes survive. Run after migrations and
-- separately from parallel integration tests (console purge visits orphans).
\set ON_ERROR_STOP on
BEGIN;

-- Exercise the actual V19 file as Flyway's restricted owner, against legacy
-- rows in a disposable pre-V19 schema. An empty migrated DB cannot catch an
-- UPDATE silently skipped by FORCE RLS.
SELECT 's5_production_' || replace(gen_random_uuid()::text, '-', '') AS fixture_schema \gset
CREATE SCHEMA :"fixture_schema" AUTHORIZATION tallerflow_migrator;
SET LOCAL search_path TO :"fixture_schema", public;
SET LOCAL ROLE tallerflow_migrator;
CREATE TABLE workshops (id uuid PRIMARY KEY);
CREATE TABLE users (id uuid PRIMARY KEY);
CREATE TABLE orders (id uuid PRIMARY KEY, workshop_id uuid REFERENCES workshops(id) ON DELETE CASCADE);
CREATE TABLE order_stages (
    id uuid PRIMARY KEY, workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    order_id uuid REFERENCES orders(id) ON DELETE CASCADE,
    position smallint CHECK (position BETWEEN 1 AND 7), name text NOT NULL
);
CREATE TABLE stage_photos (stage_position smallint CHECK (stage_position BETWEEN 1 AND 7));
INSERT INTO workshops VALUES ('10000000-0000-0000-0000-000000000001'), ('10000000-0000-0000-0000-000000000002');
INSERT INTO users VALUES ('10000000-0000-0000-0000-000000000003');
INSERT INTO orders VALUES ('10000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000001');
INSERT INTO order_stages VALUES
 ('10000000-0000-0000-0000-000000000005', '10000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000004', 1, 'CORTE'),
 ('10000000-0000-0000-0000-000000000006', '10000000-0000-0000-0000-000000000002', NULL, 1, 'PERSONALIZADA');
ALTER TABLE order_stages ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_stages FORCE ROW LEVEL SECURITY;
CREATE POLICY stages_tenant ON order_stages TO tallerflow_app
 USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);
GRANT SELECT ON order_stages TO tallerflow_app;
DO $$ BEGIN
 IF EXISTS (SELECT FROM pg_roles WHERE rolname = current_user AND (rolsuper OR rolbypassrls)) THEN
  RAISE EXCEPTION 'Migration test must use restricted Flyway role';
 END IF;
 IF EXISTS (SELECT FROM order_stages) THEN
  RAISE EXCEPTION 'Legacy fixtures must be hidden from migrator before V19';
 END IF;
END $$;

\ir ../migrations/V19__production_configuration.sql

RESET ROLE;
DO $$ BEGIN
 IF (SELECT count(*) FROM order_stages WHERE client_description <> '') <> 2 THEN
  RAISE EXCEPTION 'V19 skipped legacy descriptions across tenants under FORCE RLS';
 END IF;
 IF NOT EXISTS (SELECT FROM order_stages WHERE name = 'CORTE' AND client_description = 'Preparamos y cortamos la tela de tus prendas.') THEN
  RAISE EXCEPTION 'Known legacy stage did not receive its customer description';
 END IF;
 IF NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid = 'order_stages'::regclass) THEN
  RAISE EXCEPTION 'V19 must restore FORCE RLS after backfill';
 END IF;
END $$;
INSERT INTO order_stages (id, workshop_id, order_id, position, name, client_description)
 VALUES ('10000000-0000-0000-0000-000000000007', '10000000-0000-0000-0000-000000000001',
 '10000000-0000-0000-0000-000000000004', 20, 'FINAL', 'Última etapa');
INSERT INTO stage_photos (stage_position) VALUES (20);
DO $$ BEGIN
 BEGIN
  INSERT INTO stage_photos (stage_position) VALUES (21);
  RAISE EXCEPTION 'Evidence position 21 must be rejected';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
END $$;
GRANT USAGE ON SCHEMA :"fixture_schema" TO tallerflow_app;
INSERT INTO production_settings (workshop_id, stages) VALUES
 ('10000000-0000-0000-0000-000000000001', '[{"name":"Nueva etapa","client_description":"Nuevo proceso"}]'),
 ('10000000-0000-0000-0000-000000000002', '[{"name":"Privada","client_description":"Otro taller"}]');
SET LOCAL ROLE tallerflow_app;
SELECT set_config('app.workshop_id', '10000000-0000-0000-0000-000000000001', true);
DO $$ BEGIN
 IF (SELECT count(*) FROM production_settings) <> 1 OR (SELECT count(*) FROM order_stages) <> 2 THEN
  RAISE EXCEPTION 'Tenant isolation was weakened by the migration';
 END IF;
 IF NOT EXISTS (SELECT FROM order_stages WHERE name = 'CORTE') THEN
  RAISE EXCEPTION 'Saving a template must preserve historical stage snapshots';
 END IF;
 BEGIN
  INSERT INTO production_settings (workshop_id, stages)
  VALUES ('10000000-0000-0000-0000-000000000002', '[{}]');
  RAISE EXCEPTION 'Cross-tenant settings write was accepted';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $$;

-- Now exercise the REAL console function and public FKs, not a mock DELETE.
RESET ROLE;
SET LOCAL search_path TO public;
SELECT gen_random_uuid() AS purge_workshop, gen_random_uuid() AS keep_workshop,
 gen_random_uuid() AS departed_actor, gen_random_uuid() AS keeper,
 gen_random_uuid() AS purge_client, gen_random_uuid() AS keep_client,
 gen_random_uuid() AS purge_order, gen_random_uuid() AS keep_order,
 gen_random_uuid() AS console_admin \gset
SELECT set_config('s5.purge_workshop', :'purge_workshop', true),
 set_config('s5.keep_workshop', :'keep_workshop', true),
 set_config('s5.departed_actor', :'departed_actor', true),
 set_config('s5.keep_order', :'keep_order', true);
INSERT INTO users (id, email, name) VALUES
 (:'departed_actor', :'departed_actor' || '@s5-contract.test', 'Former member'),
 (:'keeper', :'keeper' || '@s5-contract.test', 'Remaining owner');
INSERT INTO workshops (id, name, status, deletion_scheduled_for) VALUES
 (:'purge_workshop', 's5-purge-' || :'purge_workshop', 'PENDING_DELETION', now() - interval '1 day'),
 (:'keep_workshop', 's5-keep-' || :'keep_workshop', 'ACTIVE', NULL);
INSERT INTO workshop_members (workshop_id, user_id, role) VALUES
 (:'purge_workshop', :'departed_actor', 'OWNER'), (:'keep_workshop', :'keeper', 'OWNER');
INSERT INTO clients (id, workshop_id, name) VALUES
 (:'purge_client', :'purge_workshop', 'Deleted client'), (:'keep_client', :'keep_workshop', 'Remaining client');
INSERT INTO orders (id, workshop_id, client_id, order_number, code, product, quantity, unit, promised_at, responsible_user_id) VALUES
 (:'purge_order', :'purge_workshop', :'purge_client', 1, 'S5-1', 'Deleted order', 1, 'UNITS', now(), :'departed_actor'),
 (:'keep_order', :'keep_workshop', :'keep_client', 1, 'S5-1', 'Remaining order', 1, 'UNITS', now(), :'departed_actor');
INSERT INTO production_settings (workshop_id, stages) VALUES
 (:'purge_workshop', '[{"name":"Entrega","client_description":"Lista"}]'),
 (:'keep_workshop', '[{"name":"Corte","client_description":"En proceso"}]');
-- A former member can remain the author of another workshop's old history.
INSERT INTO order_changes (workshop_id, order_id, actor_user_id, changes) VALUES
 (:'purge_workshop', :'purge_order', :'departed_actor', '{"block_reason":{"from":"","to":"Falta tela"}}'),
 (:'keep_workshop', :'keep_order', :'departed_actor', '{"specification":{"from":"A","to":"B","reason":"Cambio acordado"}}');
INSERT INTO console_admins (id, email, password_hash, totp_secret_ciphertext)
 VALUES (:'console_admin', :'console_admin' || '@s5-contract.test', 'contract-only', decode('01', 'hex'));
SET LOCAL ROLE tallerflow_console;
SELECT console_purge_workshop(:'purge_workshop'::uuid, 's5-purge-' || :'purge_workshop', :'console_admin'::uuid, '127.0.0.1');
RESET ROLE;
DO $$ BEGIN
 IF EXISTS (SELECT FROM workshops WHERE id = current_setting('s5.purge_workshop')::uuid)
 OR EXISTS (SELECT FROM production_settings WHERE workshop_id = current_setting('s5.purge_workshop')::uuid)
 OR EXISTS (SELECT FROM order_changes WHERE workshop_id = current_setting('s5.purge_workshop')::uuid) THEN
  RAISE EXCEPTION 'Production FKs blocked or incompletely cascaded console purge';
 END IF;
 IF EXISTS (SELECT FROM users WHERE id = current_setting('s5.departed_actor')::uuid) THEN
  RAISE EXCEPTION 'Production references prevented orphan actor cleanup';
 END IF;
 IF NOT EXISTS (SELECT FROM orders WHERE id = current_setting('s5.keep_order')::uuid AND responsible_user_id IS NULL)
 OR NOT EXISTS (SELECT FROM order_changes WHERE order_id = current_setting('s5.keep_order')::uuid AND actor_user_id IS NULL AND changes ? 'specification') THEN
  RAISE EXCEPTION 'Deleting an actor must clear assignment and preserve surviving history';
 END IF;
 IF NOT EXISTS (SELECT FROM console_audit_events WHERE workshop_id = current_setting('s5.purge_workshop')::uuid AND event_type = 'WORKSHOP_PURGED') THEN
  RAISE EXCEPTION 'Console purge must retain its administrative audit record';
 END IF;
END $$;

-- An order can also be deleted independently of its workshop.
DELETE FROM orders WHERE id = :'keep_order'::uuid;
DO $$ BEGIN
 IF EXISTS (SELECT FROM order_changes WHERE order_id = current_setting('s5.keep_order')::uuid) THEN
  RAISE EXCEPTION 'Order deletion must cascade its dependent change history';
 END IF;
 IF NOT EXISTS (SELECT FROM production_settings WHERE workshop_id = current_setting('s5.keep_workshop')::uuid) THEN
  RAISE EXCEPTION 'Deleting one order must not delete its workshop settings';
 END IF;
END $$;
ROLLBACK;
