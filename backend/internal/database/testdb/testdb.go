// Package testdb gives tests a real, empty, migrated Postgres database of their own.
package testdb

import (
	"context"
	"crypto/rand"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/database"
)

// New connects to the database at TEST_DATABASE_URL and returns a pool whose tables live in a
// fresh schema with every migration applied. The schema is dropped when the test ends.
//
// Each test gets its own schema because go test runs packages in parallel against the same
// database, so shared tables would see each other's rows.
//
// Without TEST_DATABASE_URL the test is skipped, except in CI, where it fails: a test that
// silently doesn't run must never count as passing there.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI")
		}
		t.Skip("TEST_DATABASE_URL is not set; see docs/local-development.md")
	}

	ctx := context.Background()
	schema := "test_" + strings.ToLower(rand.Text())
	exec(t, url, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize())
	t.Cleanup(func() { exec(t, url, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE") })

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close) // runs before the schema is dropped: cleanups run last-in, first-out

	if _, err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return pool
}

// exec runs one statement on its own connection.
func exec(t testing.TB, url, sql string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
