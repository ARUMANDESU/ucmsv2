package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Import the stdlib driver for pgx

	"github.com/ARUMANDESU/ucms/pkg/ctxs"
	"github.com/ARUMANDESU/ucms/pkg/env"
)

func NewPgxPool(ctx context.Context, pgdsn string, mode env.Mode) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(pgdsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgdsn: %w", err)
	}

	opts := []otelpgx.Option{
		otelpgx.WithTrimSQLInSpanName(),
	}
	if mode == env.Prod {
		opts = append(opts, otelpgx.WithDisableSQLStatementInAttributes()) // disable SQL statements in attributes to avoid PII/high-cardinality
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer(opts...)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func Migrate(dsn string, fs *embed.FS) error {
	driver, err := iofs.New(fs, "migrations")
	if err != nil {
		return err
	}
	defer driver.Close()

	m, err := migrate.NewWithSourceInstance("iofs", driver, dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	return fn(ctxs.WithTx(ctx, tx), tx)
}
