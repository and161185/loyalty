package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/and161185/loyalty/internal/config"
	"github.com/and161185/loyalty/internal/model"
)

func TestGetStatus_OK(t *testing.T) {
	accrual := 50.5
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"order":   "1234567890",
			"status":  "PROCESSED",
			"accrual": accrual,
		})
	}))
	defer ts.Close()

	cfg := &config.Config{AccuralSystemAddress: ts.URL}
	srv := &Server{config: cfg}
	order := model.Order{Number: "1234567890"}

	updated, err := srv.getStatus(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Status != model.Processed {
		t.Errorf("expected status %s, got %s", model.Processed, updated.Status)
	}

	if updated.Accrual == nil || *updated.Accrual != accrual {
		t.Errorf("expected accrual %.2f, got %v", accrual, updated.Accrual)
	}
}

func TestGetStatus_TooManyRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	cfg := &config.Config{AccuralSystemAddress: ts.URL}
	srv := &Server{config: cfg}
	order := model.Order{Number: "1234567890"}

	start := time.Now()
	_, err := srv.getStatus(context.Background(), order)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if duration < time.Second {
		t.Errorf("expected sleep of at least 1s, got %s", duration)
	}
}

func TestGetStatus_NoContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	cfg := &config.Config{AccuralSystemAddress: ts.URL}
	srv := &Server{config: cfg}
	order := model.Order{Number: "1234567890"}

	updated, err := srv.getStatus(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated != order {
		t.Errorf("expected unchanged order, got %+v", updated)
	}
}
