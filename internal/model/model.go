package model

import "time"

type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (m *User) AsIfaceList() []any {
	return []any{&m.ID, &m.TelegramID, &m.CreatedAt, &m.UpdatedAt}
}
