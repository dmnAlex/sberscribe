package repository

import (
	"context"
	"encoding/json"

	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type Repository interface {
	DoTx(f func(r Repository) error, opts ...*pgx.TxOptions) error

	GetOrCreateUser(ctx context.Context, telegramID int64) (model.User, error)

	CreateMeeting(ctx context.Context, userID int64, telegramFileID string) (int64, error)
	UpdateMeeting(ctx context.Context, id int64, title, summary string) error

	CreateTranscription(ctx context.Context, meetingID int64, requestFileID string, status model.TranscriptionStatus) error
	UpdateTranscription(ctx context.Context, meetingID int64, responseFileID, content *string, raw json.RawMessage, status model.TranscriptionStatus) error

	GetMeetinsByUser(ctx context.Context, userID int64) ([]model.Meeting, error)
	GetMeetingWithTranscription(ctx context.Context, userID, id int64) (model.MeetingWithTranscription, error)

	FindMeetings(ctx context.Context, userID int64, query string) ([]model.Meeting, error)
}

type SberScribeRepo struct {
	db *pg.DB
}

func New(db *pg.DB) Repository {
	return &SberScribeRepo{db: db}
}

func (r *SberScribeRepo) DoTx(f func(r Repository) error, opts ...*pgx.TxOptions) error {
	return r.db.DoTx(func(db *pg.DB) error {
		return f(New(db))
	}, opts...)
}

func (r *SberScribeRepo) Close() error {
	return errors.Wrap(r.db.Close(), "close")
}

const getMeetingsByUserSQL = `
	SELECT m.id, m.telegram_file_id, m.title, m.summary
	FROM meetings m
	JOIN transcriptions t ON m.id = t.meeting_id
	WHERE m.user_id = @userID AND t.status = @status
	ORDER BY m.created_at
`

func (r *SberScribeRepo) GetMeetinsByUser(ctx context.Context, userID int64) ([]model.Meeting, error) {
	args := pgx.NamedArgs{
		"userID": userID,
		"status": model.StatusDone,
	}

	return pg.QueryMany(r.db, getMeetingsByUserSQL, pg.IfaceListFunc[*model.Meeting](), args)
}

const getMeetingWithTranscriptionSQL = `
	SELECT m.id, m.telegram_file_id, m.title, m.summary, 
		t.request_file_id, t.response_file_id, t.content, t.raw, t.status
	FROM meetings m
	JOIN transcriptions t ON m.id = t.meeting_id
	WHERE m.user_id = @userID AND m.id = @id
`

func (r *SberScribeRepo) GetMeetingWithTranscription(ctx context.Context, userID, id int64) (model.MeetingWithTranscription, error) {
	args := pgx.NamedArgs{
		"userID": userID,
		"id":     id,
	}

	var meeting model.MeetingWithTranscription
	err := r.db.WithCtx(ctx).QueryRow(getMeetingWithTranscriptionSQL, args, meeting.AsIfaceList()...)
	return meeting, pg.WrapNotFound(err)
}

const findMeetingsSQL = `
	SELECT m.id, m.telegram_file_id, m.title, m.summary
	FROM meetings m
	JOIN transcriptions t ON m.id = t.meeting_id
	WHERE m.user_id = @userID AND t.status = @status AND to_tsvector('russian', content) @@ websearch_to_tsquery('russian', @query)
	ORDER BY ts_rank(to_tsvector('russian', content), websearch_to_tsquery('russian', @query)) DESC, created_at DESC
`

func (r *SberScribeRepo) FindMeetings(ctx context.Context, userID int64, query string) ([]model.Meeting, error) {
	args := pgx.NamedArgs{
		"userID": userID,
		"status": model.StatusDone,
		"query":  query,
	}

	return pg.QueryMany(r.db, findMeetingsSQL, pg.IfaceListFunc[*model.Meeting](), args)
}
