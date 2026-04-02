package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/model/errx"
	"github.com/dmnAlex/sberscribe/internal/repository"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

func (s *BotService) SetupHandlers() {
	s.bot.Handle("/start", func(c telebot.Context) error {
		_, err := s.repo.GetOrCreateUser(s.stopCtx, c.Sender().ID)
		if err != nil {
			return c.Reply("Ошибка регистрации")
		}

		return c.Reply(fmt.Sprintf("👋 Привет, %s! Я умный помощник для конспектирования.\nОтправь мне голосовое сообщение или аудиофайл, и я сделаю выжимку.\n\nКоманды:\n/list - список встреч\n/get <id> - детали встречи\n/chat <вопрос> - спросить ИИ", c.Sender().FirstName))
	})

	s.bot.Handle(telebot.OnVoice, func(c telebot.Context) error {
		msg := c.Message()
		if msg.Voice == nil {
			return nil
		}

		err := s.enqueueFile(c, msg.Voice.FileID, msg.Voice.MIME, msg.Chat.ID, msg.Sender.ID)
		return errors.Wrap(err, "enqueue file")
	})

	s.bot.Handle(telebot.OnAudio, func(c telebot.Context) error {
		msg := c.Message()
		if msg.Audio == nil {
			return nil
		}

		err := s.enqueueFile(c, msg.Audio.FileID, msg.Audio.MIME, msg.Chat.ID, msg.Sender.ID)
		return errors.Wrap(err, "enqueue file")
	})

	s.bot.Handle("/list", func(c telebot.Context) error {
		user, err := s.repo.GetOrCreateUser(s.stopCtx, c.Sender().ID)
		if err != nil {
			return errors.Wrap(err, "get or create user")
		}

		meetings, err := s.repo.GetMeetinsByUser(s.stopCtx, user.ID)
		if err != nil {
			return errors.Wrap(err, "get meetings by user")
		}

		if len(meetings) == 0 {
			return c.Reply("У вас пока нет встреч")
		}

		msg := formatMeetingsTable(meetings)
		return c.Send(msg, telebot.ModeHTML)
	})

	s.bot.Handle("/find", func(c telebot.Context) error {
		user, err := s.repo.GetOrCreateUser(s.stopCtx, c.Sender().ID)
		if err != nil {
			return errors.Wrap(err, "get or create user")
		}
		query := strings.TrimSpace(strings.TrimPrefix(c.Message().Text, "/find"))
		if query == "" {
			return c.Reply("Введите поисковый запрос после /find")
		}

		meetings, err := s.repo.FindMeetings(s.stopCtx, user.ID, query)
		if err != nil {
			return errors.Wrap(err, "find meetings")
		}

		if len(meetings) == 0 {
			return c.Reply("По вашему запросу ничего не найдено")
		}

		msg := formatMeetingsTable(meetings)
		return c.Send(msg, telebot.ModeHTML)
	})

	s.bot.Handle("/get", func(c telebot.Context) error {
		user, err := s.repo.GetOrCreateUser(s.stopCtx, c.Sender().ID)
		if err != nil {
			return errors.Wrap(err, "get or create user")
		}
		text := strings.TrimSpace(strings.TrimPrefix(c.Message().Text, "/get"))
		id, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			logger.Log.Error("parse int", "error", err)
			return c.Reply("Введите команду /get <id>")
		}

		meeting, err := s.repo.GetMeetingWithTranscription(s.stopCtx, user.ID, id)
		if err != nil {
			if errors.Is(err, errx.ErrNotFound) {
				return c.Reply("Встреча не найдена")
			}

			return errors.Wrap(err, "get meeting with transcription")
		}

		if meeting.Status != model.StatusDone {
			return c.Reply("Встреча еще в обработке")
		}

		msg := formatMeetingWithTranscription(meeting)
		return c.Send(msg, telebot.ModeHTML)
	})

	s.bot.Handle("/chat", func(c telebot.Context) error {
		text := strings.TrimSpace(strings.TrimPrefix(c.Message().Text, "/chat"))
		if text == "" {
			return c.Reply("Задайте вопрос после команды /chat")
		}
		task := BotTask{
			Type:   TaskChatQuery,
			ChatID: c.Chat().ID,
			UserID: c.Sender().ID,
			Data:   text,
		}

		if err := s.SubmitTask(task); err != nil {
			return c.Reply("Очередь занята")
		}
		return nil
	})

	s.bot.Handle("/test", func(c telebot.Context) error {
		chat := c.Chat()
		logger.Log.Debug("chat data", "data", *chat)
		return nil
	})
}

func formatMeetingsTable(meetings []model.Meeting) string {
	var sb strings.Builder

	sb.WriteString("<b>Ваши встречи</b>\n\n")
	sb.WriteString("<b>ID</b> | <b>Название</b>\n")
	sb.WriteString("-----------------------\n")

	for _, m := range meetings {
		title := "Без названия"
		if m.Title != nil {
			title = *m.Title
		}

		sb.WriteString(fmt.Sprintf("%d | %s\n", m.ID, title))
	}

	return sb.String()
}

func formatMeetingWithTranscription(meeting model.MeetingWithTranscription) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>%s</b>\n\n", *meeting.Title))
	sb.WriteString(fmt.Sprintf("<i>%s</i>\n", *meeting.Summary))

	return sb.String()
}

func (s *BotService) enqueueFile(c telebot.Context, fileID, mimeType string, chatID, userID int64) error {
	var id int64
	if err := s.repo.DoTx(func(rTx repository.Repository) error {
		user, err := rTx.GetOrCreateUser(s.stopCtx, c.Sender().ID)
		if err != nil {
			return errors.Wrap(err, "get or create user")
		}

		id, err = rTx.CreateMeeting(s.stopCtx, user.ID, fileID)
		if err != nil {
			return errors.Wrap(err, "create meeting")
		}

		return nil
	}); err != nil {
		logger.Log.Error("do tx", "error", err)
		return c.Reply("Непредвиденная ошибка")
	}

	info := fileInfo{
		meetingID: id,
		fileID:    fileID,
		mimeType:  mimeType,
	}
	task := BotTask{
		Type:   TaskProcessAudio,
		ChatID: chatID,
		UserID: userID,
		Data:   info,
	}

	if err := s.SubmitTask(task); err != nil {
		return c.Reply("Сервер перегружен, попробуйте позже")
	}

	return c.Reply(fmt.Sprintf("Встреча %d принята в обработку", id))
}
