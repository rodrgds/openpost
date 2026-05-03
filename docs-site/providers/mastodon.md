# Mastodon

Mastodon is configured per instance. Each instance you support needs its own app credentials.

## What you need

- One Mastodon app per instance
- `MASTODON_SERVERS` JSON
- Callback URL: `https://your-domain.com/api/v1/accounts/mastodon/callback`

## Example configuration

```sh
MASTODON_SERVERS='[
  {
    "name": "Personal",
    "client_id": "xxx",
    "client_secret": "yyy",
    "instance_url": "https://mastodon.social"
  }
]'
```

## Multiple instances

```sh
MASTODON_SERVERS='[
  {
    "name": "Personal",
    "client_id": "abc",
    "client_secret": "def",
    "instance_url": "https://mastodon.social"
  },
  {
    "name": "Work",
    "client_id": "ghi",
    "client_secret": "jkl",
    "instance_url": "https://fosstodon.org"
  }
]'
```

## Notes

- The current backend config default for `MASTODON_REDIRECT_URI` is `http://localhost:8080/api/v1/accounts/mastodon/callback`.
- OpenPost may show the config `name` in the UI, but the persisted provider identity is the full `instance_url`.
- The stored `instance_url` needs to stay consistent with the configured provider entry.
