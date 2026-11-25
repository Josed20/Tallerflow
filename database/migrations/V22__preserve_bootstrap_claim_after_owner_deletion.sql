-- The singleton row is the installation claim, independently of its first
-- owner's lifetime. Purging that user must clear the reference, not the claim.
ALTER TABLE bootstrap_state
    ALTER COLUMN owner_user_id DROP NOT NULL,
    DROP CONSTRAINT bootstrap_state_owner_user_id_fkey,
    ADD CONSTRAINT bootstrap_state_owner_user_id_fkey
        FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE SET NULL;

COMMENT ON COLUMN bootstrap_state.owner_user_id IS
    'Initial owner reference; nullable after user deletion. The singleton row permanently records the installation claim.';
