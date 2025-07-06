package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and161185/loyalty/internal/auth"
	"github.com/and161185/loyalty/internal/errs"
	"github.com/and161185/loyalty/internal/model"
)

type mockStorage struct {
	GetUserFunc func(ctx context.Context, id int) (model.User, error)
}

func (m *mockStorage) GetUserById(ctx context.Context, id int) (model.User, error) {
	return m.GetUserFunc(ctx, id)
}

func TestAuthMiddleware(t *testing.T) {
	auth.SetSecret("test-secret")
	validToken, _ := auth.GenerateToken(1)

	tests := []struct {
		name           string
		authHeader     string
		storage        Storage
		expectedStatus int
	}{
		{
			name:           "no header",
			authHeader:     "",
			storage:        &mockStorage{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalidtoken",
			storage:        &mockStorage{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not found",
			authHeader: "Bearer " + validToken,
			storage: &mockStorage{
				GetUserFunc: func(ctx context.Context, id int) (model.User, error) {
					return model.User{}, errs.ErrUserNotFound
				},
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "storage error",
			authHeader: "Bearer " + validToken,
			storage: &mockStorage{
				GetUserFunc: func(ctx context.Context, id int) (model.User, error) {
					return model.User{}, errors.New("some db error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "ok",
			authHeader: "Bearer " + validToken,
			storage: &mockStorage{
				GetUserFunc: func(ctx context.Context, id int) (model.User, error) {
					return model.User{ID: 1, Login: "test"}, nil
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			mw := AuthMiddleware(tt.storage)
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
