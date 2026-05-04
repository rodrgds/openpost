# Production Checklist

- [ ] Generate fresh `OPENPOST_JWT_SECRET`
- [ ] Generate fresh `OPENPOST_ENCRYPTION_KEY`
- [ ] Use secrets that are at least 32 characters long
- [ ] Set `OPENPOST_APP_URL`
- [ ] Set `OPENPOST_MEDIA_URL`
- [ ] Decide whether to set `OPENPOST_DISABLE_REGISTRATIONS=true` after creating the first admin account
- [ ] Configure reverse proxy with HTTPS
- [ ] Update provider callback URLs
- [ ] Persist `/data`
- [ ] Back up database and media
- [ ] Check `GET /api/v1/health`
