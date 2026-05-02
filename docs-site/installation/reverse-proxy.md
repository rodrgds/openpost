# Reverse Proxy

HTTPS and a stable public URL matter for provider OAuth and for Threads media publishing.

## Why it matters

- Providers validate callback URLs exactly.
- `OPENPOST_FRONTEND_URL` should match what users open in the browser.
- `OPENPOST_MEDIA_URL` must be public for Threads media publishing.

## Required app settings

- `OPENPOST_FRONTEND_URL=https://openpost.example.com`
- `OPENPOST_MEDIA_URL=https://openpost.example.com/media`

## Caddy example

```txt
openpost.example.com {
  reverse_proxy localhost:8080
}
```

## Nginx example

```nginx
server {
  listen 443 ssl http2;
  server_name openpost.example.com;

  location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
```

## Callback URLs

Update your provider apps to use your public domain:

- `https://openpost.example.com/api/v1/accounts/x/callback`
- `https://openpost.example.com/api/v1/accounts/mastodon/callback`
- `https://openpost.example.com/api/v1/accounts/linkedin/callback`
- `https://openpost.example.com/api/v1/accounts/threads/callback`

## Threads note

Threads needs the media endpoint to be publicly reachable. If `OPENPOST_MEDIA_URL` points to a private hostname or plain local path, media publishing will fail.
