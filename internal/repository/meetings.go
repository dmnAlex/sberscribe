package repository

import (
	"context"
	"time"

	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/jackc/pgx/v5"
)

const createMeetingSQL = `
	INSERT INTO meetings (user_id, telegram_file_id)
	VALUES (@userID, @telegramFileID)
	RETURNING id
`

func (r *SberScribeRepo) CreateMeeting(ctx context.Context, userID int64, telegramFileID string) (int64, error) {
	args := pgx.NamedArgs{
		"userID":         userID,
		"telegramFileID": telegramFileID,
	}

	var id int64
	err := r.db.WithCtx(ctx).QueryRow(createMeetingSQL, args, &id)
	return id, pg.WrapAlreadyExists(err)
}

const updateMeetingSQL = `
	UPDATE meetings
	SET title = @title, summary = @summary, updated_at = @now 
	WHERE id = @id
`

func (r *SberScribeRepo) UpdateMeeting(ctx context.Context, id int64, title, summary string) error {
	args := pgx.NamedArgs{
		"id":      id,
		"title":   title,
		"summary": summary,
		"now":     time.Now(),
	}
	res, err := r.db.WithCtx(ctx).Exec(updateMeetingSQL, args)
	return pg.HandleExecResult(res, err)
}
