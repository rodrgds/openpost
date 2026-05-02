# Accounts

Connected accounts are provider-specific identities inside a workspace.

## Common flow

1. Open the accounts screen.
2. Choose a provider.
3. Complete the provider auth flow.
4. Return to OpenPost and confirm the account is listed.

## Notes

- Disconnecting an account does not delete your provider app credentials from `.env`.
- Stored OAuth tokens are encrypted at rest.
- Each provider has its own callback and permission requirements.
