package bot

import (
	"strings"

	"github.com/dmnAlex/sberscribe/internal/model"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleStart(c telebot.Context) error {
	task := model.InTask{
		Type:   model.TaskStart,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
		Data:   c.Sender().FirstName,
	}
	b.inCh <- task
	return nil
}

func (b *Bot) handleList(c telebot.Context) error {
	task := model.InTask{
		Type:   model.TaskList,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
	}
	b.inCh <- task
	return nil
}

func (b *Bot) handleGet(c telebot.Context) error {
	text := strings.TrimSpace(strings.TrimPrefix(c.Text(), "/get"))
	task := model.InTask{
		Type:   model.TaskGet,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
		Data:   text,
	}
	b.inCh <- task
	return nil
}

func (b *Bot) handleFind(c telebot.Context) error {
	text := strings.TrimSpace(strings.TrimPrefix(c.Text(), "/find"))
	task := model.InTask{
		Type:   model.TaskFind,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
		Data:   text,
	}
	b.inCh <- task
	return nil
}

func (b *Bot) handleChat(c telebot.Context) error {
	text := strings.TrimSpace(strings.TrimPrefix(c.Text(), "/chat"))
	task := model.InTask{
		Type:   model.TaskChat,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
		Data:   text,
	}
	b.inCh <- task
	return nil
}

func (b *Bot) handleVoice(c telebot.Context) error {
	info := model.FileInfo{
		FileID:   c.Message().Voice.FileID,
		MimeType: c.Message().Voice.MIME,
	}
	task := model.InTask{
		Type:   model.TaskRecord,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
		Data:   info,
	}
	b.inCh <- task
	return nil
}

func (b *Bot) handleAudio(c telebot.Context) error {
	info := model.FileInfo{
		FileID:   c.Message().Audio.FileID,
		MimeType: c.Message().Audio.MIME,
	}
	task := model.InTask{
		Type:   model.TaskRecord,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
		Data:   info,
	}
	b.inCh <- task
	return nil
}
