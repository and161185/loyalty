package config

import (
	"flag"
	"os"

	"go.uber.org/zap"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccuralSystemAddress string
	Logger               *zap.SugaredLogger
}

func NewConfig() *Config {
	logCfg := zap.NewProductionConfig()
	logCfg.OutputPaths = []string{"stdout", "server.log"}

	logger := zap.Must(logCfg.Build())

	cfg := &Config{}
	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "DB connection string")
	flag.StringVar(&cfg.AccuralSystemAddress, "r", "", "Accural system address")
	flag.Parse()

	cfg.Logger = logger.Sugar()

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

	if accuralSystemAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); accuralSystemAddress != "" {
		cfg.AccuralSystemAddress = accuralSystemAddress
	}
}
