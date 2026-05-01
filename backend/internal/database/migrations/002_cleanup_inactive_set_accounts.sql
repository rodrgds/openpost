-- Remove stale social media set memberships that reference disconnected or missing accounts.

DELETE FROM social_media_set_accounts
WHERE social_account_id IN (
    SELECT id
    FROM social_accounts
    WHERE is_active = 0
)
OR social_account_id NOT IN (
    SELECT id
    FROM social_accounts
);
