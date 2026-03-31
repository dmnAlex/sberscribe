package telegram

import (
	"context"
	"time"

	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/repository"
	"github.com/dmnAlex/sberscribe/internal/salutespeech"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

const pollTimeout = 10 * time.Second

type Bot struct {
	bot    *telebot.Bot
	repo   repository.Repository
	salute *salutespeech.Client
}

func New(token string, repo repository.Repository, salute *salutespeech.Client) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: pollTimeout},
	})
	if err != nil {
		return nil, errors.Wrap(err, "new bot")
	}

	bot := &Bot{bot: b, repo: repo, salute: salute}

	b.Handle("/start", bot.handleStart)
	b.Handle(telebot.OnVoice, bot.handleVoice)

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

func (b *Bot) handleVoice(c telebot.Context) error {
	return b.enqueueFile(c, c.Message().Voice.FileID, c.Message().Voice.MIME)
}

func (b *Bot) enqueueFile(c telebot.Context, fileID, mime string) error {
	file, err := b.bot.FileByID(fileID)
	if err != nil {
		return c.Send("Не удалось получить информацию о файле")
	}

	logger.Log.Debug("got an audiofile", "file", file, "mime", mime)

	data, err := b.bot.File(&file)
	if err != nil {
		return c.Send("Не удалось скачать файл")
	}
	defer data.Close()

	text, raw, err := b.salute.Recognize(context.TODO(), data, mime)
	if err != nil {
		logger.Log.Error(err.Error())
		return c.Send("Ошибка распознавания")
	}

	logger.Log.Debug("got raw transcription", "transcription", string(raw))

	return c.Send(text)
}
