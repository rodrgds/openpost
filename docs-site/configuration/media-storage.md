# Media Storage

OpenPost stores media on the local filesystem through its `BlobStorage` abstraction.

## Key settings

- `OPENPOST_MEDIA_PATH` controls where files are stored on disk.
- `OPENPOST_MEDIA_URL` controls how those files are exposed publicly.

## Recommended production values

```sh
OPENPOST_MEDIA_PATH=/data/media
OPENPOST_MEDIA_URL=https://openpost.example.com/media
```

## Why public media URLs matter

Threads requires the backend to hand Meta a publicly reachable media URL. If OpenPost cannot expose the file publicly, Threads media publishing will fail.

## Backups

Back up the media directory together with the SQLite database. You need both for a complete restore.
