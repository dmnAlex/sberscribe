package repository

import (
	"context"
	"time"

	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

const getOrCreateUserSQL = `
	INSERT INTO users (telegram_id, created_at, updated_at)
	VALUES (@telegramID, @now, @now)
	ON CONFLICT (telegram_id)
	DO UPDATE SET updated_at = @now
	RETURNING id, telegram_id, created_at, updated_at
`

func (r *SberScribeRepo) GetOrCreateUser(ctx context.Context, telegramID int64) (model.User, error) {
	args := pgx.NamedArgs{
		"telegramID": telegramID,
		"now":        time.Now(),
	}

	var user model.User
	err := r.db.WithCtx(ctx).QueryRow(getOrCreateUserSQL, args, user.AsIfaceList()...)
	return user, errors.Wrap(err, "query row")
}
