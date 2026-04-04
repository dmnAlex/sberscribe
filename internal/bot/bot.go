package bot

import (
	"context"
	"io"
	"time"

	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

const (
	pollTimeout = 10 * time.Second
	chanSize    = 100
)

type Bot struct {
	stopCtx context.Context
	bot     *telebot.Bot
	inCh    chan model.InTask
	outCh   chan model.OutTask
}

func NewBot(ctx context.Context, token string) (*Bot, error) {
	settings := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: pollTimeout},
	}

	b, err := telebot.NewBot(settings)
	if err != nil {
		return nil, errors.Wrap(err, "new bot")
	}

	bot := &Bot{
		stopCtx: ctx,
		bot:     b,
		inCh:    make(chan model.InTask, chanSize),
		outCh:   make(chan model.OutTask, chanSize),
	}

	b.Handle("/start", bot.handleStart)
	b.Handle("/list", bot.handleList)
	b.Handle("/get", bot.handleGet)
	b.Handle("/find", bot.handleFind)
	b.Handle("/chat", bot.handleChat)
	b.Handle(telebot.OnVoice, bot.handleVoice)
	b.Handle(telebot.OnAudio, bot.handleAudio)

	return bot, nil
}

func (b *Bot) Start() {
	go b.outputWorker()
	b.bot.Start()
}

func (b *Bot) Stop() {
	b.bot.Stop()
}

func (b *Bot) InCh() <-chan model.InTask   { return b.inCh }
func (b *Bot) OutCh() chan<- model.OutTask { return b.outCh }

func (b *Bot) GetFileRC(fileID string) (io.ReadCloser, error) {
	file, err := b.bot.FileByID(fileID)
	if err != nil {
		return nil, errors.Wrap(err, "get file by id")
	}

	data, err := b.bot.File(&file)
	return data, errors.Wrap(err, "get bot file reader")
}

func (b *Bot) sendReply(task model.OutTask) {
	b.bot.Send(telebot.ChatID(task.ChatID), task.Message, task.Mode)
}

func (b *Bot) outputWorker() {
	logger.Log.Debug("start output worker")
	for {
		select {
		case <-b.stopCtx.Done():
			logger.Log.Debug("stop output worker")
			return
		case task := <-b.outCh:
			b.sendReply(task)
		}
	}
}
