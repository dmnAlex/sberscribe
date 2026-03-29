package repository

import (
	"context"

	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type Repository interface {
	GetOrCreateUser(ctx context.Context, telegramID int64) (model.User, error)
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
