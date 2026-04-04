package model

import (
	"encoding/json"
	"time"

	"gopkg.in/telebot.v3"
)

type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (m *User) AsIfaceList() []any {
	return []any{&m.ID, &m.TelegramID, &m.CreatedAt, &m.UpdatedAt}
}

type RecordStatus int

const (
	StatusInit RecordStatus = iota
	StatusUploading
	StatusUploaded
	StatusRecognizing
	StatusRecognized
	StatusPolling
	StatusPolled
	StatusDownloading
	StatusDownloaded
	StatusSummarizing
	StatusSummarized
)

func (s RecordStatus) Next() RecordStatus {
	if s >= StatusSummarized {
		return s
	}
	return s + 1
}

func (s RecordStatus) Prev() RecordStatus {
	if s <= StatusInit {
		return s
	}
	return s - 1
}

type Record struct {
	ID             int64
	UserID         int64
	ChatID         int64
	BotFileID      string
	MimeType       string
	UploadFileID   *string
	TaskID         *string
	DownloadFileID *string
	Content        *string
	Title          *string
	Summary        *string
	Raw            json.RawMessage
	Status         RecordStatus
	Attempts       int
}

func (m *Record) AsIfaceList() []any {
	return []any{
		&m.ID, &m.UserID, &m.ChatID, &m.BotFileID, &m.MimeType, &m.UploadFileID, &m.TaskID, &m.DownloadFileID,
		&m.Content, &m.Title, &m.Summary, &m.Raw, &m.Status, &m.Attempts,
	}
}

type RoleType string

const (
	SystemRole    RoleType = "system"
	UserRole      RoleType = "user"
	AssistantRole RoleType = "assistant"
	FunctionRole  RoleType = "function"
)

func (s RoleType) String() string {
	return string(s)
}

type ChatMessage struct {
	Role    RoleType
	Content string
}

type SummarizeResult struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type TaskType int

const (
	TaskStart TaskType = iota
	TaskList
	TaskGet
	TaskFind
	TaskRecord
	TaskChat
)

type InTask struct {
	Type   TaskType
	ChatID int64
	UserID int64
	Data   any
}

func NewInTask(c telebot.Context, typ TaskType, data any) InTask {
	return InTask{
		Type:   typ,
		ChatID: c.Chat().ID,
		UserID: c.Sender().ID,
		Data:   data,
	}
}

type OutTask struct {
	ChatID  int64
	Message string
	Mode    telebot.ParseMode
}

func NewOutTask(chatID int64, msg string, mode telebot.ParseMode) OutTask {
	return OutTask{
		ChatID:  chatID,
		Message: msg,
		Mode:    mode,
	}
}

type FileInfo struct {
	FileID   string
	MimeType string
}
