package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/openpost/backend/internal/services/auth"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	EmailKey  contextKey = "email"
)

func AuthMiddleware(api huma.API, authService *auth.Service) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")
		if authHeader == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "missing authorization header")
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		claims, err := authService.ValidateToken(tokenParts[1])
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "invalid or expired token")
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
