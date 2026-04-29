package bot

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/model/msgx"
	"gopkg.in/telebot.v3"
)

const (
	queryLimit  = 100
	promptLimit = 2000
)

func (b *Bot) handleStart(c telebot.Context) error {
	b.inCh <- model.NewInTask(c, model.TaskStart, c.Sender().FirstName)
	return nil
}

func (b *Bot) handleList(c telebot.Context) error {
	b.inCh <- model.NewInTask(c, model.TaskList, c.Sender().ID)
	return nil
}

func (b *Bot) handleGet(c telebot.Context) error {
	pld := strings.TrimSpace(c.Message().Payload)
	if pld == "" {
		return c.Send("Пожалуйста, укажите ID записи. Пример: /get 123")
	}
	id, err := strconv.ParseInt(pld, 10, 64)
	if err != nil || id <= 0 {
		return c.Send("❌ Неверный формат. ID должен быть положительным целым числом. Пример: /get 123")
	}

	b.inCh <- model.NewInTask(c, model.TaskGet, id)
	return nil
}

func (b *Bot) handleFind(c telebot.Context) error {
	query := strings.TrimSpace(c.Message().Payload)
	if query == "" || strings.ToLower(query) == "help" {
		return c.Send(msgx.SearchHelpMsg, telebot.ModeMarkdown)
	}

	if utf8.RuneCountInString(query) > queryLimit {
		return c.Send(fmt.Sprintf("❌ Поисковый запрос слишком длинный. Пожалуйста, сократите его до %d символов.", queryLimit))
	}

	b.inCh <- model.NewInTask(c, model.TaskFind, query)
	return nil
}

func (b *Bot) handleChat(c telebot.Context) error {
	prompt := strings.TrimSpace(c.Message().Payload)
	if prompt == "" {
		return c.Send("Пожалуйста, напишите запрос для нейросети. Пример: /chat Напиши стих про Go")
	}
	if utf8.RuneCountInString(prompt) > promptLimit {
		return c.Send(fmt.Sprintf("❌ Ваш запрос слишком длинный. Пожалуйста, ограничьтесь %d символами.", promptLimit))
	}

	b.inCh <- model.NewInTask(c, model.TaskChat, prompt)
	return nil
}

func (b *Bot) handleVoice(c telebot.Context) error {
	info := model.FileInfo{
		FileID:   c.Message().Voice.FileID,
		MimeType: c.Message().Voice.MIME,
	}
	b.inCh <- model.NewInTask(c, model.TaskRecord, info)
	return nil
}

func (b *Bot) handleAudio(c telebot.Context) error {
	info := model.FileInfo{
		FileID:   c.Message().Audio.FileID,
		MimeType: c.Message().Audio.MIME,
	}
	b.inCh <- model.NewInTask(c, model.TaskRecord, info)
	return nil
}

func (b *Bot) handleText(c telebot.Context) error {
	return c.Send(msgx.CommandListMsg, telebot.ModeMarkdown)
}
