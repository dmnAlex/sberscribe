package service

import (
	"fmt"
	"strings"

	"github.com/dmnAlex/sberscribe/internal/logger"
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

		var id int64
		if err := s.repo.DoTx(func(rTx repository.Repository) error {
			user, err := rTx.GetOrCreateUser(s.stopCtx, c.Sender().ID)
			if err != nil {
				return errors.Wrap(err, "get or create user")
			}

			id, err = rTx.CreateMeeting(s.stopCtx, user.ID, msg.Voice.FileID)
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
			fileID:    msg.Voice.FileID,
			mimeType:  msg.Voice.MIME,
		}
		task := BotTask{
			Type:   TaskProcessAudio,
			ChatID: msg.Chat.ID,
			UserID: msg.Sender.ID,
			Data:   info,
		}

		if err := s.SubmitTask(task); err != nil {
			return c.Reply("Сервер перегружен, попробуйте позже")
		}

		return c.Reply(fmt.Sprintf("Встреча %d принята в обработку", id))
	})

	s.bot.Handle("/chat", func(c telebot.Context) error {
		text := strings.TrimPrefix(c.Message().Text, "/chat ")
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
