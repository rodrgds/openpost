package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/uptrace/bun"
)

type AuthHandler struct {
	db   *bun.DB
	auth *auth.Service
}

func NewAuthHandler(db *bun.DB, authService *auth.Service) *AuthHandler {
	return &AuthHandler{db: db, auth: authService}
}

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

type UserProfile struct {
	ID        string    `json:"id" doc:"User ID"`
	Email     string    `json:"email" doc:"User email address"`
	CreatedAt time.Time `json:"created_at" doc:"Account creation time"`
}

type AuthOutput struct {
	Body struct {
		Token string       `json:"token" doc:"JWT authentication token"`
		User  *UserProfile `json:"user"`
	}
}

type MeOutput struct {
	Body *UserProfile
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
		var existingUser models.User
		err := h.db.NewSelect().Model(&existingUser).
			Where("email = ?", input.Body.Email).
			Scan(ctx)
		if err == nil {
			return nil, huma.Error409Conflict("email already registered")
		}

		passwordHash, err := h.auth.HashPassword(input.Body.Password)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to hash password")
		}

		user := &models.User{
			ID:           uuid.New().String(),
			Email:        input.Body.Email,
			PasswordHash: passwordHash,
			CreatedAt:    time.Now(),
		}

		if _, err := h.db.NewInsert().Model(user).Exec(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to create user")
		}

		token, err := h.auth.GenerateToken(user.ID, user.Email)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to generate token")
		}

		resp := &AuthOutput{}
		resp.Body.Token = token
		resp.Body.User = &UserProfile{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		}
		return resp, nil
	})
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
			Where("email = ?", input.Body.Email).
			Scan(ctx)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}

		if !h.auth.CheckPassword(input.Body.Password, user.PasswordHash) {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}

		token, err := h.auth.GenerateToken(user.ID, user.Email)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to generate token")
		}

		resp := &AuthOutput{}
		resp.Body.Token = token
		resp.Body.User = &UserProfile{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		}
		return resp, nil
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
	}, func(ctx context.Context, input *struct{}) (*MeOutput, error) {
		userID := middleware.GetUserID(ctx)

		user := new(models.User)
		err := h.db.NewSelect().Model(user).
			Where("id = ?", userID).
			Scan(ctx)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}

		return &MeOutput{
			Body: &UserProfile{
				ID:        user.ID,
				Email:     user.Email,
				CreatedAt: user.CreatedAt,
			},
		}, nil
	})
}
