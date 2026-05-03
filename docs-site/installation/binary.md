# Single Binary

OpenPost can run as a single Go binary with the frontend embedded into the executable.

## 1. Download a release

Download the binary for your platform from [GitHub Releases](https://github.com/rodrgds/openpost/releases).

## 2. Create `.env`

```bash
cp backend/.env.example .env
```

## 3. Make it executable

```bash
chmod +x ./openpost
```

## 4. Run it

```bash
./openpost
```

By default, OpenPost listens on `http://localhost:8080`.

## Notes

- Set `OPENPOST_DATABASE_PATH` and `OPENPOST_MEDIA_PATH` explicitly for production.
- Put the service behind HTTPS before enabling production OAuth callbacks.
