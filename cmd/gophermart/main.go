package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/and161185/loyalty/internal/config"
	"github.com/and161185/loyalty/internal/deps"
	"github.com/and161185/loyalty/internal/server"
	"github.com/and161185/loyalty/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config := config.NewConfig()
	deps := deps.NewDependencies(config.Key)

	storage, err := storage.NewPostgreStorage(ctx, config.DatabaseURI)
	if err != nil {
		deps.Logger.Fatal(err)
	}

	srv := server.NewServer(storage, storage, storage, config, deps)
	if err := srv.Run(ctx); err != nil {
		deps.Logger.Fatal(err)
	}
}
