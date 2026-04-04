package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type Repository interface {
	DoTx(f func(r Repository) error, opts ...*pgx.TxOptions) error

	GetOrCreateUser(ctx context.Context, telegramID int64) (model.User, error)

	CreateRecord(userID, chatID int64, botFileID, mimeType string) (int64, error)
	GetRecord(userID, id int64) (model.Record, error)
	GetRecordsByUserID(userID int64) ([]model.Record, error)
	FindRecords(userID int64, query string) ([]model.Record, error)

	UpdateRecordUploadFileID(ID int64, uploadFileID string) error
	UpdateRecordTaskID(ID int64, taskID string) error
	UpdateRecordDownloadFileID(ID int64, downloadFileID string) error
	UpdateRecordContentAndRaw(ID int64, content string, raw json.RawMessage) error
	UpdateRecordTitleAndSummary(ID int64, title, summary string) error

	GetNextRecordsBatch(batchSize int) ([]model.Record, error)
	RollbackRecordStatus(ID int64) error
	ReleaseStaleRecords(staleThreshold time.Time) error
	DeleteRecord(ID int64) error

	Close() error
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
