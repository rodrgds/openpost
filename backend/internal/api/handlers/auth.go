package handlers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/openpost/backend/internal/services/mfa"
	"github.com/uptrace/bun"
)

const (
	authChallengeLoginMFA     = "login_mfa"
	authChallengeTOTPSetup    = "totp_setup"
	authChallengePasskeySetup = "passkey_setup"
	authChallengePasskeyLogin = "passkey_login"
	mfaMethodTOTP             = "totp"
	mfaMethodPasskey          = "passkey"
	passwordReauthError       = "current password is required"
	defaultPasskeyDisplayName = "Unnamed passkey"
)

type AuthHandler struct {
	db                    *bun.DB
	auth                  *auth.Service
	encryptor             *crypto.TokenEncryptor
	mfa                   *mfa.Service
	registrationsDisabled bool
}

func NewAuthHandler(db *bun.DB, authService *auth.Service, encryptor *crypto.TokenEncryptor, mfaService *mfa.Service, registrationsDisabled bool) *AuthHandler {
	return &AuthHandler{
		db:                    db,
		auth:                  authService,
		encryptor:             encryptor,
		mfa:                   mfaService,
		registrationsDisabled: registrationsDisabled,
	}
}

var (
	errEmailAlreadyRegistered = errors.New("email already registered")
	errRegistrationsDisabled  = errors.New("registrations are disabled for this instance")
)

type RegisterInput struct {
	Body struct {
		Email    string `json:"email" format:"email" doc:"User email address"`
		Password string `json:"password" minLength:"8" doc:"User password (min 8 characters)"`
	}
}

type LoginInput struct {
	Body struct {
		Email    string `json:"email" format:"email" doc:"User email address"`
		Password string `json:"password" doc:"User password"`
	}
}

type VerifyTOTPLoginInput struct {
	Body struct {
		MFAToken string `json:"mfa_token" doc:"Pending MFA challenge token"`
		Code     string `json:"code" minLength:"6" maxLength:"6" doc:"Six digit authenticator code"`
	}
}

type BeginPasskeyLoginInput struct {
	Body struct {
		MFAToken string `json:"mfa_token" doc:"Pending MFA challenge token"`
	}
}

type FinishPasskeyLoginInput struct {
	Body struct {
		ChallengeID string          `json:"challenge_id" doc:"Passkey challenge ID"`
		Credential  json.RawMessage `json:"credential" doc:"WebAuthn assertion response"`
	}
}

type SetupTOTPInput struct {
	Body struct {
		CurrentPassword string `json:"current_password" doc:"Current password for re-authentication"`
	}
}

type ConfirmTOTPSetupInput struct {
	Body struct {
		ChallengeID string `json:"challenge_id" doc:"TOTP setup challenge ID"`
		Code        string `json:"code" minLength:"6" maxLength:"6" doc:"Six digit authenticator code"`
	}
}

type DisableTOTPInput struct {
	Body struct {
		CurrentPassword string `json:"current_password" doc:"Current password for re-authentication"`
	}
}

type BeginPasskeyRegistrationInput struct {
	Body struct {
		CurrentPassword string `json:"current_password" doc:"Current password for re-authentication"`
		Name            string `json:"name" doc:"Optional passkey label"`
	}
}

type FinishPasskeyRegistrationInput struct {
	Body struct {
		ChallengeID string          `json:"challenge_id" doc:"Passkey registration challenge ID"`
		Name        string          `json:"name" doc:"Optional passkey label"`
		Credential  json.RawMessage `json:"credential" doc:"WebAuthn registration response"`
	}
}

type RemovePasskeyInput struct {
	PasskeyID string `path:"passkey_id" doc:"Passkey ID"`
	Body      struct {
		CurrentPassword string `json:"current_password" doc:"Current password for re-authentication"`
	}
}

type UserProfile struct {
	ID        string    `json:"id" doc:"User ID"`
	Email     string    `json:"email" doc:"User email address"`
	CreatedAt time.Time `json:"created_at" doc:"Account creation time"`
}

type AuthOutput struct {
	Body struct {
		Token       string       `json:"token,omitempty" doc:"JWT authentication token"`
		User        *UserProfile `json:"user,omitempty"`
		RequiresMFA bool         `json:"requires_mfa" doc:"Whether the login requires a second factor"`
		MFAToken    string       `json:"mfa_token,omitempty" doc:"Pending MFA token for follow-up verification"`
		MFAMethods  []string     `json:"mfa_methods,omitempty" doc:"Enabled MFA methods for this account"`
	}
}

type MeOutput struct {
	Body *UserProfile
}

type PasskeySummary struct {
	ID         string    `json:"id" doc:"Passkey ID"`
	Name       string    `json:"name" doc:"User-visible passkey label"`
	CreatedAt  time.Time `json:"created_at" doc:"When the passkey was registered"`
	LastUsedAt time.Time `json:"last_used_at" doc:"When the passkey was last used"`
}

type SecurityStatusOutput struct {
	Body struct {
		User        *UserProfile     `json:"user"`
		TOTPEnabled bool             `json:"totp_enabled" doc:"Whether authenticator-based 2FA is enabled"`
		Passkeys    []PasskeySummary `json:"passkeys"`
		Methods     []string         `json:"methods" doc:"Currently available MFA methods"`
	}
}

type SetupTOTPOutput struct {
	Body struct {
		ChallengeID    string `json:"challenge_id"`
		ManualEntryKey string `json:"manual_entry_key"`
		OTPAuthURL     string `json:"otpauth_url"`
		QRCodeDataURL  string `json:"qr_code_data_url"`
	}
}

type PasskeyCeremonyOutput struct {
	Body struct {
		ChallengeID string      `json:"challenge_id"`
		Options     interface{} `json:"options"`
	}
}

type loginChallengePayload struct {
	Methods []string `json:"methods"`
}

type totpSetupPayload struct {
	Secret          string `json:"secret,omitempty"`
	SecretEncrypted string `json:"secret_encrypted,omitempty"`
}

type passkeyChallengePayload struct {
	SessionData string `json:"session_data"`
}

func (h *AuthHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "register",
		Method:      http.MethodPost,
		Path:        "/auth/register",
		Summary:     "Register a new user",
		Tags:        []string{"Auth"},
		Errors:      []int{400, 409},
	}, func(ctx context.Context, input *RegisterInput) (*AuthOutput, error) {
		user, err := h.registerUser(ctx, input.Body.Email, input.Body.Password)
		if errors.Is(err, errEmailAlreadyRegistered) {
			return nil, huma.Error409Conflict("email already registered")
		}
		if errors.Is(err, errRegistrationsDisabled) {
			return nil, huma.Error403Forbidden("registrations are disabled for this instance")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create user")
		}

		return h.issueAuthResponse(user)
	})
}

func (h *AuthHandler) registerUser(ctx context.Context, email, password string) (*models.User, error) {
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	passwordHash, err := h.auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Email:        normalizedEmail,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	err = h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		userCount, err := tx.NewSelect().Model((*models.User)(nil)).Count(txCtx)
		if err != nil {
			return err
		}
		if h.registrationsDisabled && userCount > 0 {
			return errRegistrationsDisabled
		}

		exists, err := tx.NewSelect().Model((*models.User)(nil)).
			Where("email = ?", normalizedEmail).
			Exists(txCtx)
		if err != nil {
			return err
		}
		if exists {
			return errEmailAlreadyRegistered
		}

		user.IsAdmin = userCount == 0
		_, err = tx.NewInsert().Model(user).Exec(txCtx)
		return err
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *AuthHandler) Login(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Summary:     "Login with email and password",
		Tags:        []string{"Auth"},
		Errors:      []int{401},
	}, func(ctx context.Context, input *LoginInput) (*AuthOutput, error) {
		user := new(models.User)
		err := h.db.NewSelect().Model(user).
			Where("email = ?", strings.TrimSpace(strings.ToLower(input.Body.Email))).
			Scan(ctx)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}

		if !h.auth.CheckPassword(input.Body.Password, user.PasswordHash) {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}

		methods, err := h.enabledMFAMethods(ctx, user)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load account security")
		}
		if len(methods) == 0 {
			return h.issueAuthResponse(user)
		}

		challengeID, err := h.createChallenge(ctx, user.ID, authChallengeLoginMFA, loginChallengePayload{
			Methods: methods,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create login challenge")
		}

		resp := &AuthOutput{}
		resp.Body.RequiresMFA = true
		resp.Body.MFAToken = challengeID
		resp.Body.MFAMethods = methods
		return resp, nil
	})
}

func (h *AuthHandler) VerifyTOTPLogin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "verify-login-totp",
		Method:      http.MethodPost,
		Path:        "/auth/login/totp",
		Summary:     "Complete MFA login with a TOTP code",
		Tags:        []string{"Auth"},
		Errors:      []int{400, 401},
	}, func(ctx context.Context, input *VerifyTOTPLoginInput) (*AuthOutput, error) {
		challenge, err := h.getChallenge(ctx, input.Body.MFAToken, authChallengeLoginMFA)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid or expired MFA token")
		}

		user, err := h.getUserByID(ctx, challenge.UserID)
		if err != nil {
			return nil, huma.Error401Unauthorized("user not found")
		}

		methods, err := h.enabledMFAMethods(ctx, user)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load account security")
		}
		if !slices.Contains(methods, mfaMethodTOTP) {
			return nil, huma.Error400BadRequest("authenticator app is not enabled for this account")
		}

		secret, err := h.encryptor.Decrypt(user.TOTPSecretEnc)
		if err != nil || !h.mfa.ValidateTOTP(secret, input.Body.Code) {
			return nil, huma.Error401Unauthorized("invalid authenticator code")
		}

		if err := h.deleteChallenge(ctx, challenge.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to finish MFA login")
		}

		return h.issueAuthResponse(user)
	})
}

func (h *AuthHandler) BeginPasskeyLogin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "begin-login-passkey",
		Method:      http.MethodPost,
		Path:        "/auth/login/passkey/options",
		Summary:     "Begin MFA login with a passkey",
		Tags:        []string{"Auth"},
		Errors:      []int{400, 401},
	}, func(ctx context.Context, input *BeginPasskeyLoginInput) (*PasskeyCeremonyOutput, error) {
		challenge, err := h.getChallenge(ctx, input.Body.MFAToken, authChallengeLoginMFA)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid or expired MFA token")
		}

		user, err := h.getUserByID(ctx, challenge.UserID)
		if err != nil {
			return nil, huma.Error401Unauthorized("user not found")
		}

		passkeys, err := h.listPasskeys(ctx, user.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load passkeys")
		}
		if len(passkeys) == 0 {
			return nil, huma.Error400BadRequest("no passkeys registered for this account")
		}

		webAuthnUser, err := mfa.NewWebAuthnUser(user, passkeys)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to prepare passkey login")
		}

		options, session, err := h.mfa.BeginPasskeyLogin(webAuthnUser)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to begin passkey login")
		}

		sessionData, err := mfa.MarshalSessionData(session)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to persist passkey challenge")
		}

		passkeyChallengeID, err := h.createChallenge(ctx, user.ID, authChallengePasskeyLogin, passkeyChallengePayload{
			SessionData: sessionData,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create passkey challenge")
		}

		resp := &PasskeyCeremonyOutput{}
		resp.Body.ChallengeID = passkeyChallengeID
		resp.Body.Options = options
		return resp, nil
	})
}

func (h *AuthHandler) FinishPasskeyLogin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "finish-login-passkey",
		Method:      http.MethodPost,
		Path:        "/auth/login/passkey/verify",
		Summary:     "Complete MFA login with a passkey",
		Tags:        []string{"Auth"},
		Errors:      []int{400, 401},
	}, func(ctx context.Context, input *FinishPasskeyLoginInput) (*AuthOutput, error) {
		challenge, err := h.getChallenge(ctx, input.Body.ChallengeID, authChallengePasskeyLogin)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid or expired passkey challenge")
		}

		user, err := h.getUserByID(ctx, challenge.UserID)
		if err != nil {
			return nil, huma.Error401Unauthorized("user not found")
		}

		var payload passkeyChallengePayload
		if err := json.Unmarshal([]byte(challenge.Payload), &payload); err != nil {
			return nil, huma.Error500InternalServerError("failed to read passkey challenge")
		}

		sessionData, err := mfa.UnmarshalSessionData(payload.SessionData)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to restore passkey challenge")
		}

		passkeys, err := h.listPasskeys(ctx, user.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load passkeys")
		}

		webAuthnUser, err := mfa.NewWebAuthnUser(user, passkeys)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to prepare passkey validation")
		}

		credential, err := h.mfa.FinishPasskeyLogin(webAuthnUser, *sessionData, input.Body.Credential)
		if err != nil {
			return nil, huma.Error401Unauthorized("passkey verification failed")
		}

		if err := h.markPasskeyUsed(ctx, user.ID, credential.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to update passkey state")
		}
		if err := h.deleteChallenge(ctx, challenge.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to finish passkey login")
		}

		return h.issueAuthResponse(user)
	})
}

func (h *AuthHandler) Me(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-me",
		Method:      http.MethodGet,
		Path:        "/auth/me",
		Summary:     "Get current authenticated user",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, _ *struct{}) (*MeOutput, error) {
		userID := middleware.GetUserID(ctx)

		user, err := h.getUserByID(ctx, userID)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}

		return &MeOutput{Body: toUserProfile(user)}, nil
	})
}

func (h *AuthHandler) SecurityStatus(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-security-status",
		Method:      http.MethodGet,
		Path:        "/auth/security",
		Summary:     "Get account security settings",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, _ *struct{}) (*SecurityStatusOutput, error) {
		userID := middleware.GetUserID(ctx)
		user, err := h.getUserByID(ctx, userID)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}

		passkeys, err := h.listPasskeys(ctx, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load passkeys")
		}

		methods, err := h.enabledMFAMethods(ctx, user)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load security methods")
		}

		resp := &SecurityStatusOutput{}
		resp.Body.User = toUserProfile(user)
		resp.Body.TOTPEnabled = len(user.TOTPSecretEnc) > 0
		resp.Body.Passkeys = toPasskeySummaries(passkeys)
		resp.Body.Methods = methods
		return resp, nil
	})
}

func (h *AuthHandler) BeginTOTPSetup(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "begin-totp-setup",
		Method:      http.MethodPost,
		Path:        "/auth/security/totp/setup",
		Summary:     "Start TOTP enrollment for the current user",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 401, 409},
	}, func(ctx context.Context, input *SetupTOTPInput) (*SetupTOTPOutput, error) {
		if strings.TrimSpace(input.Body.CurrentPassword) == "" {
			return nil, huma.Error400BadRequest(passwordReauthError)
		}

		userID := middleware.GetUserID(ctx)
		user, err := h.getUserByID(ctx, userID)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}
		if len(user.TOTPSecretEnc) > 0 {
			return nil, huma.Error409Conflict("authenticator app is already enabled")
		}
		if !h.auth.CheckPassword(input.Body.CurrentPassword, user.PasswordHash) {
			return nil, huma.Error401Unauthorized("invalid current password")
		}

		key, qrPNG, err := h.mfa.GenerateTOTP(user.Email)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to generate authenticator secret")
		}

		secretEnc, err := h.encryptor.Encrypt(key.Secret())
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to protect authenticator secret")
		}

		challengeID, err := h.createChallenge(ctx, user.ID, authChallengeTOTPSetup, totpSetupPayload{
			SecretEncrypted: base64.StdEncoding.EncodeToString(secretEnc),
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create setup challenge")
		}

		resp := &SetupTOTPOutput{}
		resp.Body.ChallengeID = challengeID
		resp.Body.ManualEntryKey = key.Secret()
		resp.Body.OTPAuthURL = key.URL()
		resp.Body.QRCodeDataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)
		return resp, nil
	})
}

func (h *AuthHandler) ConfirmTOTPSetup(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "confirm-totp-setup",
		Method:      http.MethodPost,
		Path:        "/auth/security/totp/confirm",
		Summary:     "Confirm TOTP enrollment with a verification code",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 401},
	}, func(ctx context.Context, input *ConfirmTOTPSetupInput) (*SecurityStatusOutput, error) {
		challenge, err := h.getChallenge(ctx, input.Body.ChallengeID, authChallengeTOTPSetup)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid or expired setup challenge")
		}
		if challenge.UserID != middleware.GetUserID(ctx) {
			return nil, huma.Error401Unauthorized("invalid setup challenge")
		}

		var payload totpSetupPayload
		if err := json.Unmarshal([]byte(challenge.Payload), &payload); err != nil {
			return nil, huma.Error500InternalServerError("failed to read setup challenge")
		}

		secret, err := h.resolveTOTPSetupSecret(payload)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to read setup challenge")
		}
		if !h.mfa.ValidateTOTP(secret, input.Body.Code) {
			return nil, huma.Error400BadRequest("invalid authenticator code")
		}

		secretEnc, err := h.encryptor.Encrypt(secret)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to save authenticator secret")
		}

		if _, err := h.db.NewUpdate().Model((*models.User)(nil)).
			Set("totp_secret_encrypted = ?", secretEnc).
			Set("totp_enabled_at = ?", time.Now().UTC()).
			Where("id = ?", challenge.UserID).
			Exec(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to enable authenticator app")
		}
		if err := h.deleteChallenge(ctx, challenge.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to finish setup")
		}

		return h.securityStatusResponse(ctx, challenge.UserID)
	})
}

func (h *AuthHandler) DisableTOTP(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "disable-totp",
		Method:      http.MethodPost,
		Path:        "/auth/security/totp/disable",
		Summary:     "Disable TOTP for the current user",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 401},
	}, func(ctx context.Context, input *DisableTOTPInput) (*SecurityStatusOutput, error) {
		if strings.TrimSpace(input.Body.CurrentPassword) == "" {
			return nil, huma.Error400BadRequest(passwordReauthError)
		}

		userID := middleware.GetUserID(ctx)
		user, err := h.getUserByID(ctx, userID)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}
		if !h.auth.CheckPassword(input.Body.CurrentPassword, user.PasswordHash) {
			return nil, huma.Error401Unauthorized("invalid current password")
		}

		if _, err := h.db.NewUpdate().Model((*models.User)(nil)).
			Set("totp_secret_encrypted = NULL").
			Set("totp_enabled_at = NULL").
			Where("id = ?", userID).
			Exec(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to disable authenticator app")
		}

		return h.securityStatusResponse(ctx, userID)
	})
}

func (h *AuthHandler) BeginPasskeyRegistration(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "begin-passkey-registration",
		Method:      http.MethodPost,
		Path:        "/auth/security/passkeys/begin",
		Summary:     "Begin passkey registration for the current user",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 401},
	}, func(ctx context.Context, input *BeginPasskeyRegistrationInput) (*PasskeyCeremonyOutput, error) {
		if strings.TrimSpace(input.Body.CurrentPassword) == "" {
			return nil, huma.Error400BadRequest(passwordReauthError)
		}

		userID := middleware.GetUserID(ctx)
		user, err := h.getUserByID(ctx, userID)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}
		if !h.auth.CheckPassword(input.Body.CurrentPassword, user.PasswordHash) {
			return nil, huma.Error401Unauthorized("invalid current password")
		}

		passkeys, err := h.listPasskeys(ctx, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load passkeys")
		}

		webAuthnUser, err := mfa.NewWebAuthnUser(user, passkeys)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to prepare passkey registration")
		}

		options, session, err := h.mfa.BeginPasskeyRegistration(webAuthnUser)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to begin passkey registration")
		}

		sessionData, err := mfa.MarshalSessionData(session)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to persist passkey registration")
		}

		challengeID, err := h.createChallenge(ctx, userID, authChallengePasskeySetup, passkeyChallengePayload{
			SessionData: sessionData,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create passkey registration challenge")
		}

		resp := &PasskeyCeremonyOutput{}
		resp.Body.ChallengeID = challengeID
		resp.Body.Options = options
		return resp, nil
	})
}

func (h *AuthHandler) FinishPasskeyRegistration(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "finish-passkey-registration",
		Method:      http.MethodPost,
		Path:        "/auth/security/passkeys/finish",
		Summary:     "Finish passkey registration for the current user",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 401},
	}, func(ctx context.Context, input *FinishPasskeyRegistrationInput) (*SecurityStatusOutput, error) {
		challenge, err := h.getChallenge(ctx, input.Body.ChallengeID, authChallengePasskeySetup)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid or expired passkey challenge")
		}
		if challenge.UserID != middleware.GetUserID(ctx) {
			return nil, huma.Error401Unauthorized("invalid passkey challenge")
		}

		user, err := h.getUserByID(ctx, challenge.UserID)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}

		var payload passkeyChallengePayload
		if err := json.Unmarshal([]byte(challenge.Payload), &payload); err != nil {
			return nil, huma.Error500InternalServerError("failed to read passkey challenge")
		}

		sessionData, err := mfa.UnmarshalSessionData(payload.SessionData)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to restore passkey challenge")
		}

		passkeys, err := h.listPasskeys(ctx, user.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load passkeys")
		}

		webAuthnUser, err := mfa.NewWebAuthnUser(user, passkeys)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to prepare passkey registration")
		}

		credential, err := h.mfa.FinishPasskeyRegistration(webAuthnUser, *sessionData, input.Body.Credential)
		if err != nil {
			return nil, huma.Error400BadRequest(fmt.Sprintf("passkey registration failed: %s", err.Error()))
		}

		credentialJSON, err := json.Marshal(credential)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to save passkey")
		}

		name := strings.TrimSpace(input.Body.Name)
		if name == "" {
			name = defaultPasskeyDisplayName
		}

		record := &models.UserPasskey{
			ID:             uuid.New().String(),
			UserID:         user.ID,
			Name:           name,
			CredentialID:   credential.ID,
			CredentialJSON: string(credentialJSON),
			CreatedAt:      time.Now().UTC(),
		}

		if _, err := h.db.NewInsert().Model(record).Exec(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to store passkey")
		}
		if _, err := h.db.NewUpdate().Model((*models.User)(nil)).
			Set("passkey_enabled_at = ?", time.Now().UTC()).
			Where("id = ?", user.ID).
			Exec(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to update account security")
		}
		if err := h.deleteChallenge(ctx, challenge.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to finish passkey registration")
		}

		return h.securityStatusResponse(ctx, user.ID)
	})
}

func (h *AuthHandler) RemovePasskey(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "remove-passkey",
		Method:      http.MethodPost,
		Path:        "/auth/security/passkeys/{passkey_id}/remove",
		Summary:     "Remove a passkey from the current user",
		Tags:        []string{"Auth"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 401, 404},
	}, func(ctx context.Context, input *RemovePasskeyInput) (*SecurityStatusOutput, error) {
		if strings.TrimSpace(input.Body.CurrentPassword) == "" {
			return nil, huma.Error400BadRequest(passwordReauthError)
		}

		userID := middleware.GetUserID(ctx)
		user, err := h.getUserByID(ctx, userID)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}
		if !h.auth.CheckPassword(input.Body.CurrentPassword, user.PasswordHash) {
			return nil, huma.Error401Unauthorized("invalid current password")
		}

		result, err := h.db.NewDelete().Model((*models.UserPasskey)(nil)).
			Where("id = ? AND user_id = ?", input.PasskeyID, userID).
			Exec(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to remove passkey")
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return nil, huma.Error404NotFound("passkey not found")
		}

		var remaining int
		remaining, err = h.db.NewSelect().Model((*models.UserPasskey)(nil)).
			Where("user_id = ?", userID).
			Count(ctx)
		if err == nil && remaining == 0 {
			_, _ = h.db.NewUpdate().Model((*models.User)(nil)).
				Set("passkey_enabled_at = NULL").
				Where("id = ?", userID).
				Exec(ctx)
		}

		return h.securityStatusResponse(ctx, userID)
	})
}

func (h *AuthHandler) getUserByID(ctx context.Context, userID string) (*models.User, error) {
	user := new(models.User)
	if err := h.db.NewSelect().Model(user).Where("id = ?", userID).Scan(ctx); err != nil {
		return nil, err
	}
	return user, nil
}

func (h *AuthHandler) issueAuthResponse(user *models.User) (*AuthOutput, error) {
	token, err := h.auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to generate token")
	}

	resp := &AuthOutput{}
	resp.Body.Token = token
	resp.Body.User = toUserProfile(user)
	return resp, nil
}

func (h *AuthHandler) enabledMFAMethods(ctx context.Context, user *models.User) ([]string, error) {
	methods := make([]string, 0, 2)
	if len(user.TOTPSecretEnc) > 0 {
		methods = append(methods, mfaMethodTOTP)
	}

	count, err := h.db.NewSelect().Model((*models.UserPasskey)(nil)).
		Where("user_id = ?", user.ID).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		methods = append(methods, mfaMethodPasskey)
	}
	return methods, nil
}

func (h *AuthHandler) createChallenge(ctx context.Context, userID, challengeType string, payload interface{}) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	record := &models.AuthChallenge{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      challengeType,
		Payload:   string(payloadBytes),
		ExpiresAt: mfa.ChallengeExpiry(),
		CreatedAt: time.Now().UTC(),
	}
	if _, err := h.db.NewInsert().Model(record).Exec(ctx); err != nil {
		return "", err
	}
	return record.ID, nil
}

func (h *AuthHandler) getChallenge(ctx context.Context, challengeID, challengeType string) (*models.AuthChallenge, error) {
	challenge := new(models.AuthChallenge)
	err := h.db.NewSelect().Model(challenge).
		Where("id = ? AND type = ?", challengeID, challengeType).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(challenge.ExpiresAt) {
		_ = h.deleteChallenge(ctx, challenge.ID)
		return nil, fmt.Errorf("challenge expired")
	}
	return challenge, nil
}

func (h *AuthHandler) deleteChallenge(ctx context.Context, challengeID string) error {
	_, err := h.db.NewDelete().Model((*models.AuthChallenge)(nil)).Where("id = ?", challengeID).Exec(ctx)
	return err
}

func (h *AuthHandler) resolveTOTPSetupSecret(payload totpSetupPayload) (string, error) {
	if payload.SecretEncrypted != "" {
		secretEnc, err := base64.StdEncoding.DecodeString(payload.SecretEncrypted)
		if err != nil {
			return "", err
		}
		return h.encryptor.Decrypt(secretEnc)
	}
	return payload.Secret, nil
}

func (h *AuthHandler) listPasskeys(ctx context.Context, userID string) ([]models.UserPasskey, error) {
	var passkeys []models.UserPasskey
	if err := h.db.NewSelect().Model(&passkeys).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []models.UserPasskey{}, nil
		}
		return nil, err
	}
	return passkeys, nil
}

func (h *AuthHandler) markPasskeyUsed(ctx context.Context, userID string, credentialID []byte) error {
	_, err := h.db.NewUpdate().Model((*models.UserPasskey)(nil)).
		Set("last_used_at = ?", time.Now().UTC()).
		Where("user_id = ? AND credential_id = ?", userID, credentialID).
		Exec(ctx)
	return err
}

func (h *AuthHandler) securityStatusResponse(ctx context.Context, userID string) (*SecurityStatusOutput, error) {
	user, err := h.getUserByID(ctx, userID)
	if err != nil {
		return nil, huma.Error404NotFound("user not found")
	}

	passkeys, err := h.listPasskeys(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load passkeys")
	}

	methods, err := h.enabledMFAMethods(ctx, user)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load account security")
	}

	resp := &SecurityStatusOutput{}
	resp.Body.User = toUserProfile(user)
	resp.Body.TOTPEnabled = len(user.TOTPSecretEnc) > 0
	resp.Body.Passkeys = toPasskeySummaries(passkeys)
	resp.Body.Methods = methods
	return resp, nil
}

func toUserProfile(user *models.User) *UserProfile {
	return &UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

func toPasskeySummaries(passkeys []models.UserPasskey) []PasskeySummary {
	items := make([]PasskeySummary, 0, len(passkeys))
	for _, passkey := range passkeys {
		items = append(items, PasskeySummary{
			ID:         passkey.ID,
			Name:       passkey.Name,
			CreatedAt:  passkey.CreatedAt,
			LastUsedAt: passkey.LastUsedAt,
		})
	}
	return items
}
