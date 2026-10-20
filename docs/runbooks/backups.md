# Encrypted PostgreSQL backups

Production data lives on the Contabo VPS. The backup destination is a **private Cloudflare R2 bucket** in the team's Cloudflare account. An AWS account is not needed; AWS CLI speaks R2's S3-compatible API. Never use the photos bucket for database backups.

## Prepare the destination

1. Create a private R2 bucket dedicated to backups. Create an R2 token scoped to this bucket with object read/write access. Keep the access key and secret on the VPS only, in `/etc/tallerflow/backup.env` (mode `0600`, owner `root`).
2. On an offline workstation, generate an age identity with `age-keygen -o /secure/path/tallerflow-backup-identity.txt`. Give only its public recipient (`age1...`) to the VPS. Keep the identity in two separately controlled secure copies, including one held by Jose. Losing it makes every encrypted backup unreadable.
3. Install `age` and AWS CLI v2 on the VPS. Configure the following environment file there, replacing the placeholders locally:

   ```dotenv
   BACKUP_AGE_RECIPIENT=age1...
   BACKUP_S3_BUCKET=tallerflow-backups
   AWS_ENDPOINT_URL=https://<cloudflare-account-id>.r2.cloudflarestorage.com
   AWS_ACCESS_KEY_ID=<bucket-scoped-key>
   AWS_SECRET_ACCESS_KEY=<bucket-scoped-secret>
   AWS_DEFAULT_REGION=auto
   ```

4. Install `infra/systemd/tallerflow-backup.service` and `.timer` into `/etc/systemd/system/`; then run `sudo systemctl daemon-reload`. Leave the timer disabled until the first backup succeeds.
5. Run one controlled backup with `sudo systemctl start tallerflow-backup.service`. Check `journalctl -u tallerflow-backup.service -n 50` and the corresponding daily object in R2. `backup-vps.sh` verifies that the upload can be read with `head-object` before reporting success.
6. Run the isolated restore drill below. Once it passes, enable the timer with `sudo systemctl enable --now tallerflow-backup.timer` and check `systemctl list-timers tallerflow-backup.timer`.

The script streams `pg_dump` through age encryption; it never writes an unencrypted dump to disk. Daily objects use `daily/YYYY-MM-DD.dump.age`; Sundays also use `weekly/YYYY-Www.dump.age`. Set lifecycle expiration on `daily/` after 7 days and `weekly/` after 28 days in the R2 bucket, then verify the rules in Cloudflare. This retains about seven daily and four weekly recovery points; record the oldest available objects periodically. Keep another independent copy of the age identity.

## Restore drill

Before R2 is available, run `bash scripts/restore-drill-local.sh` from the repository root after starting Docker Compose (on Windows, use Git Bash). This checks the dump format locally, but not the external backup.

For the real drill, use a separate workstation or test host with Docker, `age`, AWS CLI, the offline age identity, and a bucket-scoped **read** token. Set `BACKUP_S3_BUCKET`, `AWS_ENDPOINT_URL`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION=auto` and `BACKUP_AGE_IDENTITY=/secure/path/tallerflow-backup-identity.txt` in that host's environment. Run:

```bash
bash scripts/restore-drill-r2.sh daily/YYYY-MM-DD.dump.age
```

It downloads the encrypted object, decrypts it into a PostgreSQL 18 container with no network, creates the non-login roles referenced by row-level security policies, restores it, verifies the public tables and key row counts, reports elapsed seconds, and destroys the temporary container and encrypted download. It does not connect to the production database. Log the drill date, object key, duration and result outside the repo. Compare the reported counts with the source database through an authorized read-only check if exact data reconciliation is required. A backup is operational only after this external drill passes.
