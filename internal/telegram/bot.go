package telegram

import (
	"context"
	"time"

	"github.com/dmnAlex/sberscribe/internal/repository"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

const pollTimeout = 10 * time.Second

type Bot struct {
	bot  *telebot.Bot
	repo repository.Repository
}

func New(token string, repo repository.Repository) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: pollTimeout},
	})
	if err != nil {
		return nil, errors.Wrap(err, "new bot")
	}

	bot := &Bot{bot: b, repo: repo}

	b.Handle("/start", bot.handleStart)

	return bot, nil
}

func (b *Bot) Start() { b.bot.Start() }
func (b *Bot) Stop()  { b.bot.Stop() }

func (b *Bot) handleStart(c telebot.Context) error {
	_, err := b.repo.GetOrCreateUser(context.Background(), c.Sender().ID)
	if err != nil {
		return c.Send("Ошибка регистрации")
	}
	return c.Send("👋 Добро пожаловать! Загружайте аудио/голосовые сообщения.")
}
