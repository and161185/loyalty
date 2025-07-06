package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/and161185/loyalty/internal/auth"
	"github.com/and161185/loyalty/internal/config"
	"github.com/and161185/loyalty/internal/errs"
	"github.com/and161185/loyalty/internal/middleware"
	"github.com/and161185/loyalty/internal/model"
	"github.com/and161185/loyalty/internal/utils"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type Storage interface {
	CreateUser(ctx context.Context, login, passwordHash string) error
	GetUserByLogin(ctx context.Context, login string) (model.User, string, error)
	GetUserById(ctx context.Context, id int) (model.User, error)

	AddOrder(ctx context.Context, user model.User, order model.Order) (int, error)
	GetUserOrders(ctx context.Context, user model.User) ([]model.Order, error)

	GetUserBalance(ctx context.Context, user model.User) (model.Balance, error)
	WithdrawBalance(ctx context.Context, user model.User, order string, sum float64) error
	GetWithdrawals(ctx context.Context, user model.User) ([]model.Withdrawal, error)

	GetUnprocessedOrders(ctx context.Context) ([]model.Order, error)
	UpdateOrder(ctx context.Context, order model.Order) error
}

type Server struct {
	storage Storage
	config  *config.Config
}

func NewServer(storage Storage, config *config.Config) *Server {
	return &Server{
		storage: storage,
		config:  config,
	}
}

func (srv *Server) buildRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(chiMiddleware.StripSlashes)
	router.Use(middleware.LogMiddleware(srv.config.Logger))
	router.Use(middleware.DecompressMiddleware)
	router.Use(middleware.CompressMiddleware)

	router.Post("/api/user/register", srv.RegisterHandler)
	router.Post("/api/user/login", srv.LoginHandler)

	// авторизованные ручки
	router.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(srv.storage))

		r.Post("/api/user/orders", srv.UploadOrderHandler)
		r.Get("/api/user/orders", srv.GetOrdersHandler)
		r.Get("/api/user/balance", srv.GetBalanceHandler)
		r.Post("/api/user/balance/withdraw", srv.WithdrawHandler)
		r.Get("/api/user/withdrawals", srv.GetWithdrawalsHandler)
	})

	return router
}

func (srv *Server) Run(ctx context.Context) error {
	router := srv.buildRouter()

	server := &http.Server{
		Addr:    srv.config.RunAddress,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srv.config.Logger.Fatalf("server error: %v", err)
		}
	}()

	go srv.OrdersStatusControl(ctx)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "hash error", http.StatusInternalServerError)
		return
	}

	err = s.storage.CreateUser(r.Context(), creds.Login, string(hash))
	if err != nil {
		if errors.Is(err, errs.ErrLoginAlreadyExists) {
			http.Error(w, "login taken", http.StatusConflict)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	user, _, err := s.storage.GetUserByLogin(r.Context(), creds.Login)
	if err != nil {
		http.Error(w, "failed to fetch user", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds model.Credentials

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	user, hash, err := s.storage.GetUserByLogin(r.Context(), creds.Login)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(creds.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) UploadOrderHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(model.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	number := strings.TrimSpace(string(body))
	if !utils.IsValidLuhn(number) {
		http.Error(w, "invalid order format", http.StatusUnprocessableEntity)
		return
	}

	order := model.Order{Number: number}
	code, err := s.storage.AddOrder(r.Context(), user, order)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)
}

func (s *Server) GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(model.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := s.storage.GetUserOrders(r.Context(), user)
	if err != nil {
		http.Error(w, "failed to get orders", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func (s *Server) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(model.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := s.storage.GetUserBalance(r.Context(), user)
	if err != nil {
		http.Error(w, "failed to get balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balance); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func (s *Server) WithdrawHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(model.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Order == "" || req.Sum <= 0 || !utils.IsValidLuhn(req.Order) {
		http.Error(w, "invalid input", http.StatusUnprocessableEntity)
		return
	}

	err := s.storage.WithdrawBalance(r.Context(), user, req.Order, req.Sum)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInsufficientFunds):
			http.Error(w, "insufficient funds", http.StatusPaymentRequired)
		default:
			http.Error(w, "withdraw failed", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetWithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(model.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := s.storage.GetWithdrawals(r.Context(), user)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}
