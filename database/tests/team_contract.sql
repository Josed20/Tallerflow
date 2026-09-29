BEGIN;

INSERT INTO users (id, email, name, status)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'owner-team-a@example.test', 'Owner A', 'ACTIVE'),
    ('10000000-0000-0000-0000-000000000002', 'operator-team-a@example.test', 'Operator A', 'ACTIVE');

INSERT INTO workshops (id, name)
VALUES ('20000000-0000-0000-0000-000000000001', 'Team A');

INSERT INTO workshop_members (id, workshop_id, user_id, role, status)
VALUES
    ('30000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'OWNER', 'ACTIVE'),
    ('30000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', 'OPERATOR', 'ACTIVE');

INSERT INTO team_invitations (workshop_id, email, role, token_hash, created_by, expires_at)
VALUES ('20000000-0000-0000-0000-000000000001', 'new-user@example.test', 'OPERATOR', decode(repeat('01', 32), 'hex'), '10000000-0000-0000-0000-000000000001', now() + interval '7 days');

DO $$
BEGIN
    INSERT INTO team_invitations (workshop_id, email, role, token_hash, created_by, expires_at)
    VALUES ('20000000-0000-0000-0000-000000000001', 'new-user@example.test', 'OPERATOR', decode(repeat('02', 32), 'hex'), '10000000-0000-0000-0000-000000000001', now() + interval '7 days');
    RAISE EXCEPTION 'duplicate active invitation was accepted';
EXCEPTION WHEN unique_violation THEN
END $$;

DO $$
BEGIN
    INSERT INTO workshop_members (workshop_id, user_id, role, status)
    VALUES (gen_random_uuid(), '10000000-0000-0000-0000-000000000002', 'OPERATOR', 'ACTIVE');
    RAISE EXCEPTION 'duplicate user membership was accepted';
EXCEPTION WHEN unique_violation THEN
END $$;

ROLLBACK;
