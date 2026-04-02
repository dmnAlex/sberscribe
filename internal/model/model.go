package model

import (
	"encoding/json"
	"slices"
	"time"
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

type Meeting struct {
	ID             int64   `json:"id"`
	TelegramFileID string  `json:"telegram_file_id"`
	Title          *string `json:"title,omitempty"`
	Summary        *string `json:"summary,omitempty"`
}

func (m *Meeting) AsIfaceList() []any {
	return []any{&m.ID, &m.TelegramFileID, &m.Title, &m.Summary}
}

type TranscriptionStatus string

const (
	StatusUndefined TranscriptionStatus = "UNDEFINED"
	StatusNew       TranscriptionStatus = "NEW"
	StatusRunning   TranscriptionStatus = "RUNNING"
	StatusCanceled  TranscriptionStatus = "CANCELED"
	StatusDone      TranscriptionStatus = "DONE"
	StatusError     TranscriptionStatus = "ERROR"
)

type Transcription struct {
	RequestFileID  string              `json:"request_file_id"`
	ResponseFileID *string             `json:"response_file_id"`
	Content        *string             `json:"content"`
	Raw            json.RawMessage     `json:"raw,omitempty"`
	Status         TranscriptionStatus `json:"status"`
}

func (m *Transcription) AsIfaceList() []any {
	return []any{&m.RequestFileID, &m.ResponseFileID, &m.Content, &m.Raw, &m.Status}
}

type MeetingWithTranscription struct {
	Meeting
	Transcription
}

func (m *MeetingWithTranscription) AsIfaceList() []any {
	return slices.Concat(m.Meeting.AsIfaceList(), m.Transcription.AsIfaceList())
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
