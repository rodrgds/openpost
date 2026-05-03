package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/stretchr/testify/require"
)

func TestRegisterUserMakesFirstUserAdminEvenWhenRegistrationsDisabled(t *testing.T) {
	t.Parallel()

	db := createHandlerTestDB(t, (*models.User)(nil))
	handler := NewAuthHandler(db, auth.NewService("test-secret"), nil, nil, true)

	user, err := handler.registerUser(context.Background(), "admin@example.com", "password123")
	require.NoError(t, err)
	require.True(t, user.IsAdmin)
}

func TestRegisterUserRejectsAdditionalUsersWhenRegistrationsDisabled(t *testing.T) {
	t.Parallel()

	db := createHandlerTestDB(t, (*models.User)(nil))
	handler := NewAuthHandler(db, auth.NewService("test-secret"), nil, nil, true)

	_, err := handler.registerUser(context.Background(), "admin@example.com", "password123")
	require.NoError(t, err)

	_, err = handler.registerUser(context.Background(), "user@example.com", "password123")
	require.ErrorIs(t, err, errRegistrationsDisabled)
}

func TestRegisterUserOnlyPromotesTheFirstUser(t *testing.T) {
	t.Parallel()

	db := createHandlerTestDB(t, (*models.User)(nil))
	handler := NewAuthHandler(db, auth.NewService("test-secret"), nil, nil, false)

	firstUser, err := handler.registerUser(context.Background(), "admin@example.com", "password123")
	require.NoError(t, err)
	require.True(t, firstUser.IsAdmin)

	secondUser, err := handler.registerUser(context.Background(), "user@example.com", "password123")
	require.NoError(t, err)
	require.False(t, secondUser.IsAdmin)
}

func TestResolveTOTPSetupSecretDecryptsEncryptedPayload(t *testing.T) {
	t.Parallel()

	encryptor := crypto.NewTokenEncryptor("test-secret")
	handler := NewAuthHandler(nil, nil, encryptor, nil, false)

	secretEnc, err := encryptor.Encrypt("super-secret-seed")
	require.NoError(t, err)

	secret, err := handler.resolveTOTPSetupSecret(totpSetupPayload{
		SecretEncrypted: base64.StdEncoding.EncodeToString(secretEnc),
	})
	require.NoError(t, err)
	require.Equal(t, "super-secret-seed", secret)
}

func TestCreateChallengeDoesNotPersistPlaintextTOTPSecret(t *testing.T) {
	t.Parallel()

	db := createHandlerTestDB(t, (*models.AuthChallenge)(nil))
	encryptor := crypto.NewTokenEncryptor("test-secret")
	handler := NewAuthHandler(db, nil, encryptor, nil, false)
	ctx := context.Background()

	secretEnc, err := encryptor.Encrypt("super-secret-seed")
	require.NoError(t, err)

	challengeID, err := handler.createChallenge(ctx, "user-1", authChallengeTOTPSetup, totpSetupPayload{
		SecretEncrypted: base64.StdEncoding.EncodeToString(secretEnc),
	})
	require.NoError(t, err)

	challenge, err := handler.getChallenge(ctx, challengeID, authChallengeTOTPSetup)
	require.NoError(t, err)
	require.NotContains(t, challenge.Payload, "super-secret-seed")

	var payload totpSetupPayload
	require.NoError(t, json.Unmarshal([]byte(challenge.Payload), &payload))

	secret, err := handler.resolveTOTPSetupSecret(payload)
	require.NoError(t, err)
	require.Equal(t, "super-secret-seed", secret)
}
