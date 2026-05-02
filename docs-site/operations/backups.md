# Backups

You need both the database and the media directory for a usable backup.

## Database

```bash
cp /data/db/openpost.db openpost-backup-$(date +%Y%m%d).db
```

## Media

```bash
tar -czf media-backup-$(date +%Y%m%d).tar.gz /data/media/
```

## What to back up

- SQLite database
- Media directory
- Your `.env` file or secret-management equivalent

## Notes

- Test restores, not just backups.
- Keep database and media snapshots reasonably aligned in time.
