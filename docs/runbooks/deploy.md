# Deploy and rollback

The `Deploy VPS` workflow is manual. Its GitHub `production` environment must require reviewer approval. Configure `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_PATH`, `DEPLOY_SSH_KEY`, `DEPLOY_KNOWN_HOSTS`, and `TF_HEALTH_URL` as GitHub Actions secrets. `DEPLOY_KNOWN_HOSTS` contains the reviewed SSH host key for the VPS; do not use `StrictHostKeyChecking=no`.

The VPS checkout must be clean and on `main`. The deploy script fast-forwards from `origin/main`, rebuilds Compose services and checks the HTTPS readiness endpoint. If readiness fails, it checks out the prior commit, rebuilds it and reports failure. After a rollback, run `git switch main` on the VPS before the next deployment. Check `docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app ps` and the backend logs before retrying.

Only José can provide the VPS access, DNS, HTTPS endpoint and production `.env`. PostgreSQL must not publish port 5432 publicly. Verify this from outside the VPS after provisioning.
