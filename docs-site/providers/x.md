# X

## What you need

- X developer app
- `X_CLIENT_ID`
- `X_CLIENT_SECRET`
- Callback URL: `https://your-domain.com/api/v1/accounts/x/callback`

## Required scopes

- `tweet.read`
- `tweet.write`
- `users.read`
- `offline.access`

## Local development callback

```txt
http://localhost:8080/api/v1/accounts/x/callback
```

## Common errors

- Callback URL mismatch in the X developer portal
- Missing OAuth 2.0 enablement
- Wrong redirect URI override via `X_REDIRECT_URI`
