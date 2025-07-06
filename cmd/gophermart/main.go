package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/and161185/loyalty/internal/config"
	"github.com/and161185/loyalty/internal/server"
	"github.com/and161185/loyalty/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config := config.NewConfig()
	storage, err := storage.NewPostgreStorage(ctx, config.DatabaseURI)
	if err != nil {
		config.Logger.Fatal(err)
	}

	srv := server.NewServer(storage, config)
	if err := srv.Run(ctx); err != nil {
		config.Logger.Fatal(err)
	}
}
