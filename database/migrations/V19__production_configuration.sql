-- Templates affect only newly created orders; existing stages remain snapshots.
CREATE TABLE production_settings (
    workshop_id uuid PRIMARY KEY REFERENCES workshops(id) ON DELETE CASCADE,
    stages jsonb NOT NULL CHECK (jsonb_typeof(stages) = 'array' AND jsonb_array_length(stages) BETWEEN 1 AND 20),
    version bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE order_stages ADD COLUMN client_description text NOT NULL DEFAULT '';
-- Flyway owns this table, but FORCE RLS would hide every historical stage
-- from its NOBYPASSRLS role. Restore FORCE before the migration commits.
ALTER TABLE order_stages NO FORCE ROW LEVEL SECURITY;
UPDATE order_stages SET client_description = CASE name
    WHEN 'CORTE' THEN 'Preparamos y cortamos la tela de tus prendas.'
    WHEN 'COSTURA' THEN 'Confeccionamos tus prendas.'
    WHEN 'ACABADO' THEN 'Realizamos los acabados de tus prendas.'
    WHEN 'CONTROL DE CALIDAD' THEN 'Revisamos la calidad de tus prendas.'
    WHEN 'PLANCHADO' THEN 'Planchamos tus prendas.'
    WHEN 'EMPAQUE' THEN 'Empacamos tus prendas para la entrega.'
    WHEN 'ENTREGA' THEN 'Tu pedido está listo para entregar.'
    ELSE 'Tu pedido está en esta etapa de producción.' END;
ALTER TABLE order_stages FORCE ROW LEVEL SECURITY;
ALTER TABLE orders ADD COLUMN responsible_user_id uuid REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE orders ADD COLUMN block_reason text NOT NULL DEFAULT '' CHECK (length(block_reason) <= 300);
CREATE TABLE order_changes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    -- Removing an actor must not erase another order's history.
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    changes jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX order_changes_order_idx ON order_changes(workshop_id, order_id, created_at, id);
ALTER TABLE production_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE production_settings FORCE ROW LEVEL SECURITY;
ALTER TABLE order_changes ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_changes FORCE ROW LEVEL SECURITY;
CREATE POLICY production_settings_tenant ON production_settings TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);
CREATE POLICY order_changes_tenant ON order_changes TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);
GRANT SELECT, INSERT, UPDATE ON production_settings TO tallerflow_app;
GRANT SELECT, INSERT ON order_changes TO tallerflow_app;

-- Expand valid snapshot and evidence positions without rewriting old stages.
ALTER TABLE order_stages DROP CONSTRAINT order_stages_position_check;
ALTER TABLE order_stages ADD CONSTRAINT order_stages_position_check CHECK (position BETWEEN 1 AND 20);
ALTER TABLE stage_photos DROP CONSTRAINT stage_photos_stage_position_check;
ALTER TABLE stage_photos ADD CONSTRAINT stage_photos_stage_position_check CHECK (stage_position BETWEEN 1 AND 20);
