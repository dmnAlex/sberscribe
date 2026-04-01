package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dmnAlex/sberscribe/internal/gigachat"
	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/model"
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
	meetingID int64
	fileID    string
	mimeType  string
}

type BotService struct {
	stopCtx context.Context
	pool    *ants.Pool
	repo    repository.Repository
	salute  *salutespeech.Client
	giga    *gigachat.Client
	bot     *telebot.Bot
}

func NewBotService(ctx context.Context, repo repository.Repository, salute *salutespeech.Client, giga *gigachat.Client, bot *telebot.Bot) (*BotService, error) {
	pool, err := ants.NewPool(poolSize, ants.WithExpiryDuration(expiryDuration))
	if err != nil {
		return nil, errors.Wrap(err, "new pool")
	}
	return &BotService{stopCtx: ctx, pool: pool, repo: repo, salute: salute, giga: giga, bot: bot}, nil
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
		return errors.New("cast data to struct")
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

	requestFileID, err := s.salute.Upload(s.stopCtx, data)
	if err != nil {
		s.sendReply(task.ChatID, "Не удалось загрузить файл")
		return errors.Wrap(err, "salute upload")
	}

	if err := s.repo.CreateTranscription(s.stopCtx, info.meetingID, requestFileID, model.StatusNew); err != nil {
		return errors.Wrap(err, "create transcription")
	}

	// TODO продумать парсинг текста из raw
	responseFileID, content, raw, err := s.salute.Recognize(s.stopCtx, requestFileID, info.mimeType)
	if err != nil {
		s.sendReply(task.ChatID, "Ошибка распознавания файла")
		return errors.Wrap(err, "salute recognize")
	}

	logger.Log.Debug("got transcription", "content", content, "transcription", string(raw))
	if err := s.repo.UpdateTranscription(s.stopCtx, info.meetingID, &responseFileID, &content, raw, model.StatusDone); err != nil {
		return errors.Wrap(err, "update transcription")
	}

	res, err := s.summarize(content)
	if err != nil {
		return errors.Wrap(err, "summarize")
	}

	if err := s.repo.UpdateMeeting(s.stopCtx, info.meetingID, res.Title, res.Summary); err != nil {
		return errors.Wrap(err, "update meeting")
	}

	logger.Log.Debug("got summary", "title", res.Title, "summary", res.Summary)
	s.sendReply(task.ChatID, fmt.Sprintf("Обработка встречи %d завершена", info.meetingID))
	return nil
}

func (s *BotService) handleChatQuery(task BotTask) error {
	prompt, ok := task.Data.(string)
	if !ok {
		s.sendReply(task.ChatID, "Ошибка: неверный формат запроса")
		return errors.New("cast data to string")
	}

	logger.Log.Debug("got chat prompt", "prompt", prompt)
	answer, err := s.giga.Chat(s.stopCtx, []model.ChatMessage{{
		Role:    model.UserRole,
		Content: prompt,
	}})
	if err != nil {
		return errors.Wrap(err, "giga chat")
	}

	s.sendReply(task.ChatID, answer)
	return nil
}

func (s *BotService) sendReply(chatID int64, text string) {
	s.bot.Send(&telebot.Chat{ID: chatID}, text, telebot.ModeMarkdown)
}

const summarizePrompt = `
	Следующим сообщением от пользователя ты получишь транскрипцию аудио. 
	Твоя задача сформулировать название для содержимого и краткую выжимку из содержимого.
	В твоем ответе должен быть только json с двумя полями:
	- Поле title содержащее название
	- Поле summary содержащее краткую выжимку
`

func (s *BotService) summarize(content string) (model.SummarizeResult, error) {
	msgs := []model.ChatMessage{{
		Role:    model.SystemRole,
		Content: summarizePrompt,
	}}

	msgs = append(msgs, model.ChatMessage{
		Role:    model.UserRole,
		Content: content,
	})

	answer, err := s.giga.Chat(s.stopCtx, msgs)
	if err != nil {
		return model.SummarizeResult{}, errors.Wrap(err, "chat")
	}

	var res model.SummarizeResult
	if err := json.Unmarshal([]byte(answer), &res); err != nil {
		return model.SummarizeResult{}, errors.Wrap(err, "unmarshal json")
	}

	return res, nil
}
