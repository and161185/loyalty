package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/and161185/loyalty/internal/auth"
	"github.com/and161185/loyalty/internal/config"
	"github.com/and161185/loyalty/internal/deps"
	"github.com/and161185/loyalty/internal/middleware"
	"github.com/and161185/loyalty/internal/mocks"
	"github.com/and161185/loyalty/internal/model"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

func setup(t *testing.T) (*Server, *mocks.MockStorage) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockStorage := mocks.NewMockStorage(ctrl)

	logger := zaptest.NewLogger(t)
	cfg := &config.Config{}
	deps := &deps.Deps{
		TokenManager: auth.NewTokenManager("testsecret"),
		Logger:       logger.Sugar(),
	}

	srv := NewServer(mockStorage, mockStorage, mockStorage, cfg, deps)

	return srv, mockStorage
}

func newAuthenticatedRequest(method, path, token string, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestRegisterHandler(t *testing.T) {
	srv, mock := setup(t)

	mock.EXPECT().
		CreateUser(gomock.Any(), "user", gomock.Any()).
		Return(nil)

	mock.EXPECT().
		GetUserByLogin(gomock.Any(), "user").
		Return(model.User{ID: 1, Login: "user"}, "", nil)

	payload := `{"login":"user","password":"pass"}`
	req := httptest.NewRequest("POST", "/api/user/register", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.RegisterHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	authHeader := resp.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Errorf("missing token")
	}
}

func TestLoginHandler(t *testing.T) {
	srv, mock := setup(t)

	pw, _ := bcryptHash("pass")
	mock.EXPECT().
		GetUserByLogin(gomock.Any(), "user").
		Return(model.User{ID: 1, Login: "user"}, pw, nil)

	payload := `{"login":"user","password":"pass"}`
	req := httptest.NewRequest("POST", "/api/user/login", strings.NewReader(payload))
	w := httptest.NewRecorder()

	srv.LoginHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200")
	}
}

func TestUploadOrderHandler(t *testing.T) {
	srv, mock := setup(t)

	order := "12345678903"
	mock.EXPECT().
		AddOrder(gomock.Any(), gomock.Any(), model.Order{Number: order}).
		Return(http.StatusAccepted, nil)

	token, _ := srv.deps.TokenManager.GenerateToken(1)
	req := newAuthenticatedRequest("POST", "/api/user/orders", token, order)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, model.User{ID: 1})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	srv.UploadOrderHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202")
	}
}

func TestGetOrdersHandler(t *testing.T) {
	srv, mock := setup(t)

	mock.EXPECT().
		GetUserOrders(gomock.Any(), model.User{ID: 1}).
		Return([]model.Order{
			{Number: "1", Status: "PROCESSED", UploadedAt: time.Now()},
		}, nil)

	token, _ := srv.deps.TokenManager.GenerateToken(1)
	req := newAuthenticatedRequest("GET", "/api/user/orders", token, "")
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, model.User{ID: 1})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	srv.GetOrdersHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200")
	}
}

func TestGetBalanceHandler(t *testing.T) {
	srv, mock := setup(t)

	mock.EXPECT().
		GetUserBalance(gomock.Any(), model.User{ID: 1}).
		Return(model.Balance{Current: 100.0, Withdrawn: 50.0}, nil)

	token, _ := srv.deps.TokenManager.GenerateToken(1)
	req := newAuthenticatedRequest("GET", "/api/user/balance", token, "")
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, model.User{ID: 1})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	srv.GetBalanceHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200")
	}
}

func TestWithdrawHandler(t *testing.T) {
	srv, mock := setup(t)

	mock.EXPECT().
		WithdrawBalance(gomock.Any(), model.User{ID: 1}, "12345678903", 50.0).
		Return(nil)

	reqBody := `{"order":"12345678903","sum":50}`
	token, _ := srv.deps.TokenManager.GenerateToken(1)
	req := newAuthenticatedRequest("POST", "/api/user/balance/withdraw", token, reqBody)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, model.User{ID: 1})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	srv.WithdrawHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200")
	}
}

func TestGetWithdrawalsHandler(t *testing.T) {
	srv, mock := setup(t)

	mock.EXPECT().
		GetWithdrawals(gomock.Any(), model.User{ID: 1}).
		Return([]model.Withdrawal{
			{Order: "123", Sum: 10.5, ProcessedAt: time.Now()},
		}, nil)

	token, _ := srv.deps.TokenManager.GenerateToken(1)
	req := newAuthenticatedRequest("GET", "/api/user/withdrawals", token, "")
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, model.User{ID: 1})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	srv.GetWithdrawalsHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200")
	}
}

func bcryptHash(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	return string(hash), err
}
