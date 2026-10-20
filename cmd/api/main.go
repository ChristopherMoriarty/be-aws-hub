package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/auth"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/config"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/logger"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/server"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/service"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/store/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	appLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	runErr := run(cfg, appLog)
	if runErr != nil {
		appLog.Errorw("app stopped", "error", runErr)
	}
	_ = appLog.Sync()
	if runErr != nil {
		os.Exit(1)
	}
}

func run(cfg config.Config, appLog logger.Logger) error {
	authCfg, err := config.LoadAuth()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	authService := service.NewAuth(db.New(pool), auth.NewTokens(authCfg.Secret, authCfg.AccessTTL, authCfg.RefreshTTL))
	catalog := service.NewCatalog(pool)
	attempts := service.NewAttempts(pool)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.New(pool, appLog, authService, catalog, attempts),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		appLog.Infow("api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
