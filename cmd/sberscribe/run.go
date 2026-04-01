package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmnAlex/sberscribe/internal/auth"
	"github.com/dmnAlex/sberscribe/internal/config"
	"github.com/dmnAlex/sberscribe/internal/gigachat"
	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/repository"
	"github.com/dmnAlex/sberscribe/internal/salutespeech"
	"github.com/dmnAlex/sberscribe/internal/service"
	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/dmnAlex/sberscribe/internal/utils"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

const pollingTime = 10 * time.Second

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

	tokenMgr := auth.NewTokenManager(auth.NewOAuthHTTPClient(), cfg)
	tlsConfig, err := utils.NewTLSConfig(cfg.CACertPath)
	if err != nil {
		return errors.Wrap(err, "new tls config")
	}

	ssClient, err := salutespeech.NewSaluteClient(tokenMgr, tlsConfig)
	if err != nil {
		return errors.Wrap(err, "new salute client")
	}

	gcClient, err := gigachat.NewGigaClient(tokenMgr, tlsConfig, cfg.GigaChatModel)
	if err != nil {
		return errors.Wrap(err, "new giga client")
	}

	repo := repository.New(db)

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.TelegramToken,
		Poller: &telebot.LongPoller{Timeout: pollingTime},
	})
	if err != nil {
		return errors.Wrap(err, "new bot")
	}

	botService, err := service.NewBotService(globalCtx, repo, ssClient, gcClient, bot)
	if err != nil {
		return errors.Wrap(err, "new bot service")
	}
	botService.SetupHandlers()

	go bot.Start()
	logger.Log.Info("bot started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	logger.Log.Info("shutdown signal received")
	bot.Stop()

	return nil
}
