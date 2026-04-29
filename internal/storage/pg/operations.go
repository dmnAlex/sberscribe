package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

func (db *DB) WithCtx(ctx context.Context) *DB {
	return &DB{
		stopCtx: ctx,
		pool:    db.pool,
		tx:      db.tx,
	}
}

func (db *DB) DoTx(f func(*DB) error, opts ...*pgx.TxOptions) error {
	if db.tx != nil {
		return errors.Wrap(f(db), "call func with tx")
	}
	var opt *pgx.TxOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt == nil {
		opt = &pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		}
	}
	tx, err := db.pool.BeginTx(db.stopCtx, *opt)
	if err != nil {
		return errors.Wrap(err, "begin tx")
	}
	txDB := &DB{
		stopCtx: db.stopCtx,
		pool:    db.pool,
		tx:      tx,
	}
	if err := f(txDB); err != nil {
		tx.Rollback(db.stopCtx)
		return errors.Wrap(err, "call func")
	}
	return errors.Wrap(tx.Commit(db.stopCtx), "commit")
}

func (db *DB) Exec(query string, args ...any) (pgconn.CommandTag, error) {
	if db.tx != nil {
		return db.tx.Exec(db.stopCtx, query, args...)
	}
	return db.pool.Exec(db.stopCtx, query, args...)
}

func (db *DB) Query(query string, args pgx.NamedArgs, f func(pgx.Rows) error) error {
	var (
		rows pgx.Rows
		err  error
	)
	if db.tx != nil {
		rows, err = db.tx.Query(db.stopCtx, query, args)
	} else {
		rows, err = db.pool.Query(db.stopCtx, query, args)
	}
	if err != nil {
		return errors.Wrap(err, "query")
	}
	defer rows.Close()
	for rows.Next() {
		if err := f(rows); err != nil {
			return errors.Wrap(err, "do next func")
		}
	}
	return errors.Wrap(rows.Err(), "rows")
}

func (db *DB) QueryRow(query string, args pgx.NamedArgs, dest ...any) error {
	var row pgx.Row
	if db.tx != nil {
		row = db.tx.QueryRow(db.stopCtx, query, args)
	} else {
		row = db.pool.QueryRow(db.stopCtx, query, args)
	}
	return errors.Wrap(row.Scan(dest...), "scan")
}

func (db *DB) Ping() error {
	return db.pool.Ping(db.stopCtx)
}

func (db *DB) SendBatch(batch *pgx.Batch) (pgx.BatchResults, error) {
	if db.tx != nil {
		return db.tx.SendBatch(db.stopCtx, batch), nil
	}
	return db.pool.SendBatch(db.stopCtx, batch), nil
}
