# QA report: TallerFlow invitations

- Date: 2026-09-29
- Target: http://localhost:8080
- Branch: feat/s2-stephano-team
- Coverage: owner login, team invitation, invitation consumption, invited-user login, reused link
- Framework: Vue 3 frontend, Go/Gin backend, PostgreSQL, Docker Compose
- Initial health score: 95/100 (provisional for tested invitation flow)
- Final health score: 100/100 (same coverage)

## Result

QA found 2 issues and fixed both. The complete browser flow now confirms account creation, shows the invited email, carries that email to login, authenticates the new OPERATOR, and explains what to do when a link was already used.

## ISSUE-001: Registration completion did not clearly guide the invited user

- Severity: Medium
- Category: UX
- Reproduction: accept a fresh invitation, then continue to login.
- Before: success used error styling, omitted the account email, and opened an empty login form. A reused link gave no useful next step and retained the password field value.
- Fix: dedicated success state, invited email, clear password instruction, prefilled login email, recovery action for used links, and password clearing after failure.
- Fix status: verified
- Commit: 158102d
- Files: frontend/src/modules/invitations/JoinView.vue, frontend/src/modules/auth/LoginView.vue, frontend/src/styles/main.css
- Evidence: interactive browser captures inspected before and after the fix.

## ISSUE-002: Invitation API returned incompatible JSON field names

- Severity: Medium
- Category: Functional
- Reproduction: consume a fresh invitation against the real backend and inspect the success screen.
- Before: the backend serialized `Email` and `Role`; the frontend contract reads `email` and `role`, so the confirmed account email appeared blank.
- Fix: explicit lowercase JSON tags on `ConsumeResult`, protected by a Go regression test.
- Fix status: verified
- Commit: 54de9fa
- Files: backend/internal/team/domain.go, backend/internal/team/domain_regression_test.go
- Evidence: fresh invitation for `qa8.stephano@example.test` displayed the email and completed login as OPERATOR.

## Verification

- Frontend regression tests: 3 passed.
- Full frontend suite during QA: 14 passed.
- Frontend production build: passed.
- Backend test suite: passed in all packages.
- Docker Compose services: frontend, backend, gateway, and PostgreSQL healthy.
- Manual browser result: fresh invitation, confirmation, prefilled login, OPERATOR login, and reused-link recovery all passed.

PR summary: QA found 2 issues, fixed 2, health score 95 to 100 for the tested invitation flow.
