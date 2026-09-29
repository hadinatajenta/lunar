package auth

import (
	"context"
	"net/http"
	"strings"

	sharedHttp "lunar/backend/internal/shared/http"
)

type contextKey string

const userContextKey contextKey = "authenticatedUserContext"

type UserContext struct {
	UserID string
	Email  string
}

func ContextWithUser(ctx context.Context, user *UserContext) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func FromContext(ctx context.Context) (*UserContext, bool) {
	user, ok := ctx.Value(userContextKey).(*UserContext)
	return user, ok && user != nil
}

func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				sharedHttp.WriteError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				sharedHttp.WriteError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}
			tokenStr := strings.TrimSpace(parts[1])
			claims, err := ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				sharedHttp.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			userCtx := &UserContext{
				UserID: claims.UserID,
				Email:  claims.Email,
			}
			next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), userCtx)))
		})
	}
}
