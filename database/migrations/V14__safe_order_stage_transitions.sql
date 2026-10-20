-- Sprint 3: make production transitions reversible and impossible to split.

ALTER TABLE order_stages
    ADD COLUMN completed_by uuid REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE orders
    ADD COLUMN cancelled_at timestamptz,
    ADD COLUMN cancelled_by uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN cancellation_reason text;

UPDATE orders
SET cancelled_at = updated_at,
    cancellation_reason = 'Cancelada antes de Sprint 3'
WHERE status = 'CANCELLED';

ALTER TABLE orders
    ADD CONSTRAINT orders_cancellation_reason_valid CHECK (
        cancellation_reason IS NULL OR length(btrim(cancellation_reason)) BETWEEN 5 AND 300
    ),
    ADD CONSTRAINT orders_cancellation_fields_consistent CHECK (
        (status = 'CANCELLED' AND cancelled_at IS NOT NULL AND cancellation_reason IS NOT NULL)
        OR (status <> 'CANCELLED' AND cancelled_at IS NULL AND cancelled_by IS NULL AND cancellation_reason IS NULL)
    );

-- Historical data may contain more than one active stage. Keep the furthest
-- stage active and close earlier duplicates before installing the invariant.
WITH ranked AS (
    SELECT id,
           row_number() OVER (PARTITION BY order_id ORDER BY position DESC, id) AS active_rank
    FROM order_stages
    WHERE status = 'IN_PROGRESS'
)
UPDATE order_stages AS stage
SET status = 'COMPLETED',
    completed_at = COALESCE(stage.completed_at, now())
FROM ranked
WHERE stage.id = ranked.id
  AND ranked.active_rank > 1;

CREATE UNIQUE INDEX order_stages_one_in_progress_idx
    ON order_stages (order_id)
    WHERE status = 'IN_PROGRESS';

CREATE INDEX order_stages_completed_by_idx
    ON order_stages (completed_by)
    WHERE completed_by IS NOT NULL;

GRANT UPDATE (completed_by) ON order_stages TO tallerflow_app;
GRANT UPDATE (cancelled_at, cancelled_by, cancellation_reason) ON orders TO tallerflow_app;
