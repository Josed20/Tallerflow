CREATE TABLE workshop_counters (
    workshop_id uuid PRIMARY KEY REFERENCES workshops(id) ON DELETE CASCADE,
    next_order_number bigint NOT NULL DEFAULT 1 CHECK (next_order_number > 0)
);
CREATE TABLE orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    client_id uuid NOT NULL REFERENCES clients(id),
    order_number bigint NOT NULL,
    code text NOT NULL,
    product text NOT NULL CHECK (length(btrim(product)) BETWEEN 1 AND 160),
    specification text NOT NULL DEFAULT '',
    quantity numeric(12,2) NOT NULL CHECK (quantity > 0),
    unit text NOT NULL CHECK (unit IN ('UNITS','DOZENS','KILOGRAMS','METERS')),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','COMPLETED','CANCELLED')),
    promised_at timestamptz NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(workshop_id, order_number), UNIQUE(workshop_id, code)
);
CREATE TABLE order_stages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    position smallint NOT NULL CHECK (position BETWEEN 1 AND 7),
    name text NOT NULL,
    status text NOT NULL CHECK (status IN ('PENDING','IN_PROGRESS','COMPLETED')),
    completed_at timestamptz,
    UNIQUE(order_id, position)
);
CREATE TABLE idempotency_keys (
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    key text NOT NULL,
    request_hash bytea NOT NULL,
    resource_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(workshop_id, key)
);
CREATE INDEX orders_workshop_created_idx ON orders(workshop_id, created_at DESC, id DESC);
ALTER TABLE workshop_counters ENABLE ROW LEVEL SECURITY; ALTER TABLE workshop_counters FORCE ROW LEVEL SECURITY;
ALTER TABLE orders ENABLE ROW LEVEL SECURITY; ALTER TABLE orders FORCE ROW LEVEL SECURITY;
ALTER TABLE order_stages ENABLE ROW LEVEL SECURITY; ALTER TABLE order_stages FORCE ROW LEVEL SECURITY;
ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY; ALTER TABLE idempotency_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY counters_tenant ON workshop_counters TO tallerflow_app USING (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid) WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid);
CREATE POLICY orders_tenant ON orders TO tallerflow_app USING (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid) WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid);
CREATE POLICY stages_tenant ON order_stages TO tallerflow_app USING (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid) WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid);
CREATE POLICY idempotency_tenant ON idempotency_keys TO tallerflow_app USING (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid) WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id',true),'')::uuid);
GRANT SELECT,INSERT,UPDATE ON workshop_counters,orders,order_stages,idempotency_keys TO tallerflow_app;
GRANT DELETE ON idempotency_keys TO tallerflow_app;
