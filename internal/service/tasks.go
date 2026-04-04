package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/model/errx"
	"github.com/dmnAlex/sberscribe/internal/model/msgx"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

func (s *SberScribeService) taskWorker(ctx context.Context, ID int) error {
	logger.Log.Debug("start task worker", "id", ID)
	for {
		select {
		case <-ctx.Done():
			logger.Log.Debug("stop task worker", "id", ID)
			return nil
		case task := <-s.bot.InCh():
			if err := s.processTask(task); err != nil {
				logger.Log.Error("process task", "error", err)
			}
		}
	}
}

func (s *SberScribeService) processTask(task model.InTask) error {
	var (
		msg string
		err error
	)
	mode := telebot.ModeHTML
	switch task.Type {
	case model.TaskStart:
		msg, err = s.taskStart(task)
		mode = telebot.ModeMarkdown
	case model.TaskList:
		msg, err = s.taskList(task)
	case model.TaskGet:
		msg, err = s.taskGet(task)
	case model.TaskFind:
		msg, err = s.taskFind(task)
	case model.TaskRecord:
		msg, err = s.taskRecord(task)
	case model.TaskChat:
		msg, err = s.taskChat(task)
	default:
		return errors.Errorf("incorrect task type: %d", task.Type)
	}

	if err != nil {
		logger.Log.Error("process task", "error", err, "type", task.Type, "data", task.Data)
		s.bot.OutCh() <- model.NewOutTask(task.ChatID, "Что-то пошло не так.", mode)
		return nil
	}

	s.bot.OutCh() <- model.NewOutTask(task.ChatID, msg, mode)

	return nil
}

func (s *SberScribeService) taskStart(task model.InTask) (string, error) {
	name := task.Data.(string)

	_, err := s.repo.GetOrCreateUser(s.stopCtx, task.UserID)
	if err != nil {
		return "", errors.Wrap(err, "get or create user")
	}

	msg := fmt.Sprintf(msgx.GreetingMsg, name)
	return msg, nil
}

func (s *SberScribeService) taskList(task model.InTask) (string, error) {
	user, err := s.repo.GetOrCreateUser(s.stopCtx, task.UserID)
	if err != nil {
		return "", errors.Wrap(err, "get or create user")
	}

	records, err := s.repo.GetRecordsByUserID(user.ID)
	if err != nil {
		return "", errors.Wrap(err, "get meetings by user")
	}

	if len(records) == 0 {
		return "У вас пока нет встреч.", nil
	}

	return formatRecordsTable(records), nil
}

func (s *SberScribeService) taskGet(task model.InTask) (string, error) {
	id := task.Data.(int64)
	user, err := s.repo.GetOrCreateUser(s.stopCtx, task.UserID)
	if err != nil {
		return "", errors.Wrap(err, "get or create user")
	}

	record, err := s.repo.GetRecord(user.ID, id)
	if err != nil {
		if errors.Is(err, errx.ErrNotFound) {
			return "Встреча не найдена.", nil
		}

		return "", errors.Wrap(err, "get record")
	}

	if record.Status != model.StatusSummarized {
		return "Встреча еще в обработке.", nil
	}

	return formatRecord(record), nil
}

func (s *SberScribeService) taskFind(task model.InTask) (string, error) {
	query := task.Data.(string)
	user, err := s.repo.GetOrCreateUser(s.stopCtx, task.UserID)
	if err != nil {
		return "", errors.Wrap(err, "get or create user")
	}

	records, err := s.repo.FindRecords(user.ID, query)
	if err != nil {
		return "", errors.Wrap(err, "find records")
	}

	if len(records) == 0 {
		return "По вашему запросу ничего не найдено.", nil
	}

	return formatRecordsTable(records), nil
}

func (s *SberScribeService) taskRecord(task model.InTask) (string, error) {
	info := task.Data.(model.FileInfo)
	user, err := s.repo.GetOrCreateUser(s.stopCtx, task.UserID)
	if err != nil {
		return "", errors.Wrap(err, "get or create user")
	}

	id, err := s.repo.CreateRecord(user.ID, task.ChatID, info.FileID, info.MimeType)
	if err != nil {
		return "", errors.Wrap(err, "create record")
	}

	msg := fmt.Sprintf("Встреча %d принята в обработку.", id)
	return msg, nil
}

func (s *SberScribeService) taskChat(task model.InTask) (string, error) {
	prompt := task.Data.(string)
	msgs := []model.ChatMessage{{
		Role:    model.UserRole,
		Content: prompt,
	}}

	answer, err := s.giga.Chat(s.stopCtx, msgs)
	if err != nil {
		return "", errors.Wrap(err, "chat")
	}

	return answer, nil
}

func formatRecordsTable(meetings []model.Record) string {
	var sb strings.Builder

	sb.WriteString("<b>Ваши встречи</b>\n\n")
	sb.WriteString("<b>ID</b> | <b>Название</b>\n")
	sb.WriteString("-----------------------\n")

	for _, m := range meetings {
		sb.WriteString(fmt.Sprintf("%d | %s\n", m.ID, *m.Title))
	}

	return sb.String()
}

func formatRecord(record model.Record) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>%s</b>\n\n", *record.Title))
	sb.WriteString(fmt.Sprintf("<i>%s</i>\n", *record.Summary))

	return sb.String()
}
