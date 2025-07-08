package config

import (
	"testing"
)

func TestReadServerEnvironment(t *testing.T) {
	t.Setenv("RUN_ADDRESS", "127.0.0.1:9090")
	t.Setenv("DATABASE_URI", "postgres://user:pass@localhost/db")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://localhost:8088")
	t.Setenv("LOYALTY_KEY", "test-key")

	cfg := &Config{}
	ReadServerEnvironment(cfg)

	if cfg.RunAddress != "127.0.0.1:9090" {
		t.Errorf("unexpected RunAddress: got %s", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://user:pass@localhost/db" {
		t.Errorf("unexpected DatabaseURI: got %s", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://localhost:8088" {
		t.Errorf("unexpected AccuralSystemAddress: got %s", cfg.AccrualSystemAddress)
	}
	if cfg.Key != "test-key" {
		t.Errorf("unexpected loyalty key: got %s", cfg.Key)
	}
}
