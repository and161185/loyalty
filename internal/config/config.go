package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	Key                  string
}

func NewConfig() *Config {

	cfg := &Config{}
	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "DB connection string")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "Accural system address")
	flag.StringVar(&cfg.Key, "k", "default-insecure-key", "Key")
	flag.Parse()

	ReadServerEnvironment(cfg)

	return cfg
}

func ReadServerEnvironment(cfg *Config) {
	if runAddress := os.Getenv("RUN_ADDRESS"); runAddress != "" {
		cfg.RunAddress = runAddress
	}

	if databaseURI := os.Getenv("DATABASE_URI"); databaseURI != "" {
		cfg.DatabaseURI = databaseURI
	}

	if accrualSystemAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); accrualSystemAddress != "" {
		cfg.AccrualSystemAddress = accrualSystemAddress
	}

	if key := os.Getenv("LOYALTY_KEY"); key != "" {
		cfg.Key = key
	}
}
