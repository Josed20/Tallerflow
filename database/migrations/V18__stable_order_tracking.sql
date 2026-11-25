-- Legacy hashes remain valid. The first recovery creates an additional stable
-- link; only an explicit replacement/revocation invalidates existing links.
ALTER TABLE order_tracking_links ADD COLUMN recovery_nonce text
    CHECK (recovery_nonce IS NULL OR recovery_nonce ~ '^[A-Za-z0-9_-]{43}$');
DROP INDEX order_tracking_links_one_active_idx;
CREATE UNIQUE INDEX order_tracking_links_one_recoverable_idx
    ON order_tracking_links(order_id)
    WHERE revoked_at IS NULL AND recovery_nonce IS NOT NULL;

-- A dedicated timestamp avoids extending expiry when notes/photos are edited.
ALTER TABLE orders ADD COLUMN tracking_completed_at timestamptz;
-- Flyway owns the table but FORCE RLS would otherwise hide historical rows
-- from the migrator. Restore FORCE within this transactional migration.
ALTER TABLE orders NO FORCE ROW LEVEL SECURITY;
ALTER TABLE order_stages NO FORCE ROW LEVEL SECURITY;
UPDATE orders o SET tracking_completed_at = COALESCE(
    (SELECT max(s.completed_at) FROM order_stages s WHERE s.order_id = o.id),
    o.updated_at
) WHERE o.status = 'COMPLETED';
ALTER TABLE orders FORCE ROW LEVEL SECURITY;
ALTER TABLE order_stages FORCE ROW LEVEL SECURITY;

CREATE FUNCTION set_order_tracking_completion() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.status = 'COMPLETED' THEN
        IF TG_OP = 'INSERT' THEN
            NEW.tracking_completed_at := now();
        ELSIF OLD.status IS DISTINCT FROM 'COMPLETED' THEN
            NEW.tracking_completed_at := now();
        ELSE
            NEW.tracking_completed_at := OLD.tracking_completed_at;
        END IF;
    ELSIF NEW.status = 'ACTIVE' OR TG_OP = 'INSERT' THEN
        NEW.tracking_completed_at := NULL;
    ELSE
        -- Cancellation must not revive an expired completed order. Reopening
        -- is a separate ACTIVE transition and is the only reset path.
        NEW.tracking_completed_at := OLD.tracking_completed_at;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER orders_tracking_completion
    BEFORE INSERT OR UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION set_order_tracking_completion();
