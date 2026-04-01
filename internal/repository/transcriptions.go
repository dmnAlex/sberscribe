package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/jackc/pgx/v5"
)

const createTranscriptionSQL = `
	INSERT INTO transcriptions (meeting_id, request_file_id)
	VALUES (@meetingID, requestFileID)
`

func (r *SberScribeRepo) CreateTranscription(ctx context.Context, meetingID int64, requestFileID string) error {
	args := pgx.NamedArgs{
		"meetingID":     meetingID,
		"requestFileID": requestFileID,
	}

	res, err := r.db.WithCtx(ctx).Exec(createTranscriptionSQL, args)
	return pg.HandleExecResult(res, err)
}

const updateTranscriptionSQL = `
	UPDATE transcriptions
	SET response_file_id = @responseFileID,
		content = @content,
		raw = @raw,
		status = @status,
		updated_at = @now
	WHERE meeting_id = @meetingID
`

func (r *SberScribeRepo) UpdateTranscription(ctx context.Context, meetingID int64, responseFileID, content *string, raw json.RawMessage, status model.TranscriptionStatus) error {
	args := pgx.NamedArgs{
		"meetingID":      meetingID,
		"responseFileID": responseFileID,
		"content":        content,
		"raw":            raw,
		"status":         status,
		"now":            time.Now(),
	}

	res, err := r.db.WithCtx(ctx).Exec(updateTranscriptionSQL, args)
	return pg.HandleExecResult(res, err)
}
