package service

import (
	"context"
	"time"

	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/repository"
	"github.com/dmnAlex/sberscribe/internal/salutespeech"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

const (
	poolSize       = 100
	expiryDuration = 5 * time.Minute
)

type TaskType int

const (
	TaskProcessAudio TaskType = iota
	TaskChatQuery
)

type BotTask struct {
	Type   TaskType
	ChatID int64
	UserID int64
	Data   any
}

type fileInfo struct {
	fileID   string
	mimeType string
}

type BotService struct {
	stopCtx context.Context
	pool    *ants.Pool
	repo    repository.Repository
	salute  *salutespeech.Client
	bot     *telebot.Bot
}

func NewBotService(ctx context.Context, repo repository.Repository, salute *salutespeech.Client, bot *telebot.Bot) (*BotService, error) {
	pool, err := ants.NewPool(poolSize, ants.WithExpiryDuration(expiryDuration))
	if err != nil {
		return nil, errors.Wrap(err, "new pool")
	}
	return &BotService{stopCtx: ctx, pool: pool, repo: repo, salute: salute, bot: bot}, nil
}

func (s *BotService) SubmitTask(task BotTask) error {
	return s.pool.Submit(func() {
		s.processTask(task)
	})
}

func (s *BotService) processTask(task BotTask) {
	logger.Log.Debug("processing task", "type", task.Type, "user", task.UserID)

	switch task.Type {
	case TaskProcessAudio:
		if err := s.handleAudioProcessing(task); err != nil {
			logger.Log.Error("handle audio processing", "error", err)
		}
	case TaskChatQuery:
		if err := s.handleChatQuery(task); err != nil {
			logger.Log.Error("handle chat query", "error", err)
		}
	}
}

func (s *BotService) handleAudioProcessing(task BotTask) error {
	info, ok := task.Data.(fileInfo)
	if !ok {
		s.sendReply(task.ChatID, "Ошибка: неверный формат файла")
		return errors.New("cast data to string")
	}

	file, err := s.bot.FileByID(info.fileID)
	if err != nil {
		s.sendReply(task.ChatID, "Не удалось получить информацию о файле")
		return errors.Wrap(err, "get bot file by id")
	}

	logger.Log.Debug("got an audiofile", "file", file, "mime", info.mimeType)

	data, err := s.bot.File(&file)
	if err != nil {
		s.sendReply(task.ChatID, "Не удалось скачать файл")
		return errors.Wrap(err, "get bot file reader")
	}
	defer data.Close()

	// TODO продумать парсинг текста из raw
	text, raw, err := s.salute.Recognize(s.stopCtx, data, info.mimeType)
	if err != nil {
		s.sendReply(task.ChatID, "Ошибка распознавания файла")
		return errors.Wrap(err, "salute recognize")
	}

	logger.Log.Debug("got transcription", "text", text, "transcription", string(raw))
	s.sendReply(task.ChatID, text)
	return nil
}

func (s *BotService) handleChatQuery(task BotTask) error {
	s.sendReply(task.ChatID, "Функционал пока не реализован")
	return nil
}

func (s *BotService) sendReply(chatID int64, text string) {
	s.bot.Send(&telebot.Chat{ID: chatID}, text, telebot.ModeMarkdown)
}
