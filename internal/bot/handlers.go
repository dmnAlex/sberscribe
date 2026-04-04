package bot

import (
	"strings"

	"github.com/dmnAlex/sberscribe/internal/model"
	"gopkg.in/telebot.v3"
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
	text := strings.TrimSpace(strings.TrimPrefix(c.Text(), "/get"))
	b.inCh <- model.NewInTask(c, model.TaskGet, text)
	return nil
}

func (b *Bot) handleFind(c telebot.Context) error {
	text := strings.TrimSpace(strings.TrimPrefix(c.Text(), "/find"))
	b.inCh <- model.NewInTask(c, model.TaskFind, text)
	return nil
}

func (b *Bot) handleChat(c telebot.Context) error {
	text := strings.TrimSpace(strings.TrimPrefix(c.Text(), "/chat"))
	b.inCh <- model.NewInTask(c, model.TaskChat, text)
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
