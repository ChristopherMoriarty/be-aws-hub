package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/ChristopherMoriarty/be-aws-hub/internal/config"
	"github.com/ChristopherMoriarty/be-aws-hub/internal/logger"
	"github.com/ChristopherMoriarty/be-aws-hub/migrations"
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

	runErr := run(context.Background(), cfg, os.Args[1:], appLog)
	if runErr != nil {
		appLog.Errorw("migrate stopped", "error", runErr)
	}
	_ = appLog.Sync()
	if runErr != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config, args []string, appLog logger.Logger) error {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			appLog.Errorw("close database", "error", err)
		}
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	command := "up"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	return goose.RunContext(ctx, command, db, ".", args...)
}
