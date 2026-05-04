# Threads

Threads is workable, but the media URL requirement makes deployment details matter.

## What you need

- Meta developer app
- Threads API product enabled
- `THREADS_CLIENT_ID`
- `THREADS_CLIENT_SECRET`
- Callback URL: `https://your-domain.com/api/v1/accounts/threads/callback`
- Public `OPENPOST_MEDIA_URL`
- Scopes: `threads_basic`, `threads_content_publish`, `threads_manage_replies`

## Important requirement

Threads requires publicly reachable media URLs. Set:

```sh
OPENPOST_MEDIA_URL=https://your-domain.com/media
```

## Local development

For local testing, expose OpenPost through a tunnel such as ngrok so the callback URL and `/media/...` paths are publicly reachable.

## Common issues

- Media URL points at localhost
- Reverse proxy serves a different host than the callback configuration
- Meta app missing the Threads API product or scopes
