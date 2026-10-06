CREATE TABLE stage_photos (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    stage_position smallint NOT NULL CHECK (stage_position BETWEEN 1 AND 7),
    photo_data text NOT NULL,
    caption text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX stage_photos_order_idx ON stage_photos(order_id, stage_position, created_at ASC);

ALTER TABLE stage_photos ENABLE ROW LEVEL SECURITY;
ALTER TABLE stage_photos FORCE ROW LEVEL SECURITY;

CREATE POLICY stage_photos_tenant ON stage_photos TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON stage_photos TO tallerflow_app;
