# Encrypted PostgreSQL backups

Install `age` and AWS CLI on the VPS. Configure an external S3-compatible bucket (for example Cloudflare R2) and its credentials outside the repository. Set `BACKUP_AGE_RECIPIENT` to an age **public** key and `BACKUP_S3_BUCKET` to the bucket name. Keep the private age identity offline with José.

Schedule `bash scripts/backup-vps.sh /path/to/Tallerflow` daily on the VPS with systemd timer or cron. The script streams `pg_dump` through age encryption; it never writes an unencrypted dump to disk. Daily objects use `daily/YYYY-MM-DD.dump.age`; Sundays also use `weekly/YYYY-Www.dump.age`. Configure bucket lifecycle rules for 7 daily and 4 weekly copies, or an equivalent retention policy, and verify the rules in the provider console.

## Restore drill

1. Download one encrypted backup to a separate test host. Do not run a drill against production.
2. Start an isolated PostgreSQL 18 instance with the same role initialization and an empty database.
3. Decrypt with `age -d -i /secure/path/identity.txt backup.dump.age` and pipe into `pg_restore --no-owner --no-acl -U postgres -d tallerflow` on that isolated instance.
4. Check workshop, user, order and photo counts; log in to the isolated app and open an order with evidence. Record the date, backup object, elapsed time and outcome.

The backup is only considered operational after this drill passes. Configure two external monitors after José provides the domain: `/health/ready` and the landing page. Alert the group on failures and when VPS disk usage exceeds 80%.
