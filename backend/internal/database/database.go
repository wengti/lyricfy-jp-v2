// Package database connects to Postgres (Neon in production) and keeps its schema up to date
// with the migrations in migrations/.
package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// maxConns stays small: one Cloud Run instance with one user, and Neon's free compute allows
// about 100 connections in all
const maxConns = 4

// Migrations are numbered SQL files, applied in order and never edited once merged: to change
// the schema, add a new file. Embedded, so the binary carries its own schema.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Open connects to the database at url and checks that it answers. Neon suspends when idle, so
// the first connection after a quiet spell can take a few seconds.
//
// The pool pings a connection before reusing it once it has been idle for a second, so the
// connections Neon drops when it suspends are replaced rather than failing a request.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, nil
}

// Migrate applies the migrations that haven't run yet and returns how many it applied. A
// Postgres advisory lock makes concurrent callers take turns, e.g. the old and new revision
// starting together during a deploy. The lock needs a direct connection, not a pooler in
// transaction mode, which is why production uses Neon's direct connection string.
func Migrate(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return 0, fmt.Errorf("read migrations: %w", err)
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return 0, fmt.Errorf("create migration lock: %w", err)
	}

	// goose works on database/sql; this borrows connections from the pool (and doesn't close it)
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys, goose.WithSessionLocker(locker))
	if err != nil {
		return 0, fmt.Errorf("create migration provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrate: %w", err)
	}
	return len(results), nil
}
