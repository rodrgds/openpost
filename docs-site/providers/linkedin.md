# LinkedIn

LinkedIn uses OAuth 2.0 and has more approval friction than most other providers.

## What you need

- LinkedIn developer app
- `LINKEDIN_CLIENT_ID`
- `LINKEDIN_CLIENT_SECRET`
- Callback URL: `https://your-domain.com/api/v1/accounts/linkedin/callback`

## Relevant setting

```sh
OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES=false
```

If your LinkedIn app cannot obtain the permissions required for comment-style replies, set `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES=true`.

## Threading caveat

LinkedIn thread child posts are implemented as comments on the first post rather than native threaded posts.

## Common issues

- Insufficient app approval for social actions
- Callback URL mismatch
- Reply permissions missing for thread child posts
