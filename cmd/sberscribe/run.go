package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmnAlex/sberscribe/internal/config"
	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/repository"
	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/dmnAlex/sberscribe/internal/telegram"
	"github.com/pkg/errors"
)

func run() error {
	globalCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.New()
	if err != nil {
		return errors.Wrap(err, "new config")
	}

	logger.Init(cfg.LogLevel)

	db, err := pg.New(globalCtx, cfg.DatabaseDSN, cfg.MigrationsPath)
	if err != nil {
		return errors.Wrap(err, "new pg")
	}
	defer db.Close()

	repo := repository.New(db)

	bot, err := telegram.New(cfg.TelegramToken, repo)

	logger.Log.Info("starting sberscribe bot")
	go bot.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	logger.Log.Info("shutdown signal received")
	bot.Stop()

	return nil
}
