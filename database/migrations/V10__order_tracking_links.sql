-- Private, read-only links a workshop shares with its customer so they can
-- follow one order without an account. Only the SHA-256 of the token is
-- stored; generating a new link for an order revokes the previous one.
CREATE TABLE order_tracking_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id uuid NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE CHECK (length(token_hash) = 32),
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    CONSTRAINT order_tracking_links_revoked_after_creation CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE UNIQUE INDEX order_tracking_links_one_active_idx
    ON order_tracking_links (order_id)
    WHERE revoked_at IS NULL;

ALTER TABLE order_tracking_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_tracking_links FORCE ROW LEVEL SECURITY;

CREATE POLICY order_tracking_links_tenant ON order_tracking_links
    FOR ALL TO tallerflow_app
    USING (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid)
    WITH CHECK (workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid);

-- The public tracking page has no workshop context yet: it may read only the
-- active link whose hash it presents, to learn which workshop and order to open.
CREATE POLICY order_tracking_links_public_lookup ON order_tracking_links
    FOR SELECT TO tallerflow_app
    USING (
        revoked_at IS NULL
        AND token_hash = decode(nullif(current_setting('app.tracking_token_hash', true), ''), 'hex')
    );

GRANT SELECT, INSERT, UPDATE ON order_tracking_links TO tallerflow_app;
