package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/uptrace/bun"
)

type AuthHandler struct {
	db   *bun.DB
	auth *auth.Service
}

func NewAuthHandler(db *bun.DB, authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		db:   db,
		auth: authService,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *UserProfile `json:"user"`
}

type UserProfile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	req := new(RegisterRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email and password required"})
	}

	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password must be at least 8 characters"})
	}

	var existingUser models.User
	err := h.db.NewSelect().Model(&existingUser).
		Where("email = ?", req.Email).
		Scan(c.Request().Context())

	if err == nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "Email already registered"})
	}

	passwordHash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}

	if _, err := h.db.NewInsert().Model(user).Exec(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
	}

	token, err := h.auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	return c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User: &UserProfile{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	req := new(LoginRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email and password required"})
	}

	user := new(models.User)
	err := h.db.NewSelect().Model(user).
		Where("email = ?", req.Email).
		Scan(c.Request().Context())

	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	if !h.auth.CheckPassword(req.Password, user.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	token, err := h.auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User: &UserProfile{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID := c.Get("user_id").(string)

	user := new(models.User)
	err := h.db.NewSelect().Model(user).
		Where("id = ?", userID).
		Scan(c.Request().Context())

	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, &UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}
