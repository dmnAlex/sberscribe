package pg

import (
	"iter"

	"github.com/dmnAlex/sberscribe/internal/model/errx"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

var errBreak = errors.New("break iter")

func QueryMany[T any](db *DB, query string, pointer func(*T) []any, args ...pgx.NamedArgs) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var arg pgx.NamedArgs
		if len(args) > 0 {
			arg = args[0]
		}

		err := db.Query(query, arg, func(rows pgx.Rows) error {
			var elem T
			if err := rows.Scan(pointer(&elem)...); err != nil {
				yield(elem, errors.Wrap(err, "scan"))
				return errBreak
			}

			if !yield(elem, nil) {
				return errBreak
			}

			return nil
		})

		if err != nil && !errors.Is(err, errBreak) {
			var empty T
			yield(empty, errors.Wrap(err, "query"))
		}
	}
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
