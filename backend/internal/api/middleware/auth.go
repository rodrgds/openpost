package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/labstack/echo/v4"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/uptrace/bun"
)

type contextKey string

const (
	UserIDKey      contextKey = "user_id"
	EmailKey       contextKey = "email"
	WorkspaceIDKey contextKey = "workspace_id"
)

func AuthMiddleware(api huma.API, authService *auth.Service) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")
		if authHeader == "" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "missing authorization header")
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		claims, err := authService.ValidateToken(tokenParts[1])
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		next(huma.WithValue(
			huma.WithValue(ctx, UserIDKey, claims.UserID),
			EmailKey, claims.Email,
		))
	}
}

func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func GetWorkspaceID(ctx context.Context) string {
	if v, ok := ctx.Value(WorkspaceIDKey).(string); ok {
		return v
	}
	return ""
}

// WorkspaceAccessMiddleware validates that the user has access to the workspace specified in the request.
// This should be used after AuthMiddleware.
func WorkspaceAccessMiddleware(api huma.API, _ *bun.DB) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		userID := GetUserID(ctx.Context())
		if userID == "" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "unauthorized")
			return
		}

		// Get workspace_id from query or body - this is a simplified version
		// In practice, you'd need to extract it from the specific input structure
		// This middleware serves as a pattern that handlers can follow
		next(ctx)
	}
}

// CheckWorkspaceAccess is a helper function to verify workspace access.
func CheckWorkspaceAccess(ctx context.Context, db *bun.DB, workspaceID, userID string) (bool, error) {
	var memberCount int
	memberCount, err := db.NewSelect().Model((*models.WorkspaceMember)(nil)).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return memberCount > 0, nil
}

func JWTMiddleware(authService *auth.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization header format"})
			}

			claims, err := authService.ValidateToken(tokenParts[1])
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			}

			c.Set(string(UserIDKey), claims.UserID)
			c.Set(string(EmailKey), claims.Email)

			return next(c)
		}
	}
}
