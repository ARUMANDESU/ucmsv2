package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Import the stdlib driver for pgx

	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
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
	defer func() {
		if cerr := driver.Close(); cerr != nil {
			slog.Error("failed to close migration driver", slog.String("error", cerr.Error()))
		}
	}()

	m, err := migrate.NewWithSourceInstance("iofs", driver, dsn)
	if err != nil {
		return err
	}
	defer func() {
		source, database := m.Close()
		if source != nil {
			slog.Error("failed to close migration source", slog.String("error", source.Error()))
		}
		if database != nil {
			slog.Error("failed to close migration database", slog.String("error", database.Error()))
		}
	}()

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

	fnerr := fn(ctx, tx)
	if fnerr != nil && !errorx.IsPersistable(fnerr) {
		rollbackerr := tx.Rollback(ctx)
		if rollbackerr != nil {
			slog.ErrorContext(ctx, "failed to rollback transaction", slog.String("error", rollbackerr.Error()))
			return fmt.Errorf("failed to rollback transaction: %w", rollbackerr)
		}
		slog.ErrorContext(ctx, "transaction rolled back due to error", slog.String("error", fnerr.Error()))
		return fnerr
	}

	err = tx.Commit(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to commit transaction", slog.String("error", err.Error()))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if fnerr != nil && errorx.IsPersistable(fnerr) {
		slog.DebugContext(ctx, "update function returned an error but is allowed to continue", slog.String("error", fnerr.Error()))
		return fnerr
	}

	return nil
}
