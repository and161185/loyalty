package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/and161185/loyalty/internal/auth"
	"github.com/and161185/loyalty/internal/errs"
	"github.com/and161185/loyalty/internal/model"
)

type Storage interface {
	GetUserByID(ctx context.Context, id int) (model.User, error)
}

type contextKey string

const UserContextKey contextKey = "user"

func AuthMiddleware(store Storage, tm *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			userID, err := tm.ParseToken(tokenStr)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := store.GetUserByID(r.Context(), userID)
			if err != nil {
				if err == errs.ErrUserNotFound {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
