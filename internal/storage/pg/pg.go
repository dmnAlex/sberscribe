package pg

import (
	"context"

	"github.com/pkg/errors"

	"github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	stopCtx context.Context
	pool    *pgxpool.Pool
	tx      pgx.Tx
}

func New(ctx context.Context, dsn, migrationsPath string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, errors.Wrap(err, "create new pool")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.Wrap(err, "ping pool")
	}
	if migrationsPath != "" {
		if err := applyMigrations(pool, migrationsPath); err != nil {
			pool.Close()
			return nil, errors.Wrap(err, "apply migrations")
		}
	}
	return &DB{stopCtx: ctx, pool: pool}, nil
}

func applyMigrations(pool *pgxpool.Pool, migrationsPath string) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	driver, err := pgxdriver.WithInstance(sqlDB, &pgxdriver.Config{})
	if err != nil {
		return errors.Wrap(err, "get driver with instance")
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "postgres", driver)
	if err != nil {
		return errors.Wrap(err, "migrate with db instance")
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return errors.Wrap(err, "up migrations")
	}
	return nil
}

func (db *DB) Close() error {
	if db.tx != nil {
		db.tx.Rollback(db.stopCtx)
	}
	db.pool.Close()
	return nil
}
