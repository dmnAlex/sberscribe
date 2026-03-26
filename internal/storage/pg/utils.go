package pg

import (
	"github.com/dmnAlex/sberscribe/internal/model/errx"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

func QueryMany[T any](db *DB, query string, pointer func(*T) []any, args ...pgx.NamedArgs) ([]T, error) {
	var res = []T{}
	var arg pgx.NamedArgs
	if len(args) > 0 {
		arg = args[0]
	}

	err := db.Query(query, arg, func(rows pgx.Rows) error {
		var elem T
		if err := rows.Scan(pointer(&elem)...); err != nil {
			return errors.Wrap(err, "scan")
		}

		res = append(res, elem)
		return nil
	})

	return res, errors.Wrap(err, "query")
}

type IfaceLister interface {
	AsIfaceList() []any
}

func IfaceListFunc[T IfaceLister]() func(T) []any {
	return func(t T) []any {
		return t.AsIfaceList()
	}
}

func WrapNotFound(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return errx.ErrNotFound
	}

	return errors.Wrap(err, "query")
}

func WrapAlreadyExists(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		err = errx.ErrAlreadyExists
	}

	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.InvalidTextRepresentation {
		return errors.Wrap(errx.ErrUnprocessable, err.Error())
	}
	return errors.Wrap(err, "exec")
}

func HandleExecResult(res pgconn.CommandTag, err error) error {
	if err = WrapAlreadyExists(err); err != nil {
		return errors.Wrap(err, "exec")
	}

	if res.RowsAffected() == 0 {
		return errors.Wrap(errx.ErrNotFound, "check affected")
	}
	return nil
}
