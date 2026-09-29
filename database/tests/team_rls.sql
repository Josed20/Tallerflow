BEGIN;

INSERT INTO users (id, email, name, status)
VALUES
    ('10000000-0000-0000-0000-000000000011', 'owner-team-rls-a@example.test', 'Owner A', 'ACTIVE'),
    ('10000000-0000-0000-0000-000000000012', 'owner-team-rls-b@example.test', 'Owner B', 'ACTIVE');

INSERT INTO workshops (id, name)
VALUES
    ('20000000-0000-0000-0000-000000000011', 'Team RLS A'),
    ('20000000-0000-0000-0000-000000000012', 'Team RLS B');

INSERT INTO workshop_members (id, workshop_id, user_id, role, status)
VALUES
    ('30000000-0000-0000-0000-000000000011', '20000000-0000-0000-0000-000000000011', '10000000-0000-0000-0000-000000000011', 'OWNER', 'ACTIVE'),
    ('30000000-0000-0000-0000-000000000012', '20000000-0000-0000-0000-000000000012', '10000000-0000-0000-0000-000000000012', 'OWNER', 'ACTIVE');

INSERT INTO team_invitations (workshop_id, email, role, token_hash, created_by, expires_at)
VALUES
    ('20000000-0000-0000-0000-000000000011', 'a@example.test', 'OPERATOR', decode(repeat('03', 32), 'hex'), '10000000-0000-0000-0000-000000000011', now() + interval '7 days'),
    ('20000000-0000-0000-0000-000000000012', 'b@example.test', 'OPERATOR', decode(repeat('04', 32), 'hex'), '10000000-0000-0000-0000-000000000012', now() + interval '7 days');

SET ROLE tallerflow_app;

SELECT set_config('app.workshop_id', '20000000-0000-0000-0000-000000000011', true);

DO $$
DECLARE visible_count integer;
BEGIN
    SELECT count(*) INTO visible_count FROM team_invitations;
    IF visible_count <> 1 THEN
        RAISE EXCEPTION 'expected one visible invitation, got %', visible_count;
    END IF;
END $$;

RESET ROLE;
ROLLBACK;
