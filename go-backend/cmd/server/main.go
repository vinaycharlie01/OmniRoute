package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omniroute/go-backend/internal/infrastructure/config"
	"github.com/omniroute/go-backend/internal/infrastructure/container"
	"github.com/omniroute/go-backend/internal/infrastructure/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	log, err := logger.New(cfg.LogLevel, cfg.Env)
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	defer log.Sync()

	log.Info("starting OmniRoute Go backend",
		zap.String("port", cfg.Port),
		zap.String("env", cfg.Env),
		zap.String("db", cfg.DBPath),
	)

	c, err := container.Build(cfg, log)
	if err != nil {
		log.Fatal("failed to build container", zap.Error(err))
	}
	defer c.DB.Close()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := c.Server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	log.Info("server ready", zap.String("addr", ":"+cfg.Port))
	<-quit
	log.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Server.Shutdown(ctx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}
	log.Info("server stopped")
}
