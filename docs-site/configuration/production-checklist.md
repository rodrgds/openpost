# Production Checklist

- [ ] Generate fresh `JWT_SECRET`
- [ ] Generate fresh `ENCRYPTION_KEY`
- [ ] Set `OPENPOST_ENV=production`
- [ ] Set `OPENPOST_FRONTEND_URL`
- [ ] Set `OPENPOST_MEDIA_URL`
- [ ] Configure reverse proxy with HTTPS
- [ ] Update provider callback URLs
- [ ] Persist `/data`
- [ ] Back up database and media
- [ ] Check `GET /api/v1/health`
