package repository_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/database/testdb"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

// These tests run against a real Postgres (TEST_DATABASE_URL); see the testdb package.

// Postgres stores microseconds, so test times stay whole seconds to compare exactly
var testNow = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

func hash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func session(token string, expiresAt time.Time) model.Session {
	return model.Session{
		TokenHash: hash(token),
		UserEmail: "me@example.com",
		CreatedAt: testNow.Add(-time.Hour),
		ExpiresAt: expiresAt,
	}
}

func newRepo(t *testing.T) *repository.PostgresSessionRepository {
	return repository.NewPostgresSessionRepository(testdb.New(t))
}

func mustCreate(t *testing.T, repo repository.SessionRepository, s model.Session) {
	t.Helper()
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestPostgresSessionRepository_CreateThenFind(t *testing.T) {
	repo := newRepo(t)
	want := session("token", testNow.Add(time.Hour))
	mustCreate(t, repo, want)

	got, err := repo.FindValid(context.Background(), hash("token"), testNow)
	if err != nil {
		t.Fatalf("FindValid: %v", err)
	}
	if !bytes.Equal(got.TokenHash, want.TokenHash) || got.UserEmail != want.UserEmail ||
		!got.CreatedAt.Equal(want.CreatedAt) || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("FindValid = %+v, want %+v", got, want)
	}
}

func TestPostgresSessionRepository_FindValid_NotFound(t *testing.T) {
	repo := newRepo(t)
	mustCreate(t, repo, session("token", testNow.Add(time.Hour)))

	tests := []struct {
		name  string
		token string
		now   time.Time
	}{
		{"unknown token", "other-token", testNow},
		{"expired", "token", testNow.Add(2 * time.Hour)},
		{"expiring this instant", "token", testNow.Add(time.Hour)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.FindValid(context.Background(), hash(tt.token), tt.now)
			if !errors.Is(err, repository.ErrSessionNotFound) {
				t.Fatalf("err = %v, want ErrSessionNotFound", err)
			}
		})
	}
}

func TestPostgresSessionRepository_Delete(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	mustCreate(t, repo, session("logged-out", testNow.Add(time.Hour)))
	mustCreate(t, repo, session("still-in", testNow.Add(time.Hour)))

	if err := repo.Delete(ctx, hash("logged-out")); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := repo.FindValid(ctx, hash("logged-out"), testNow); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Errorf("deleted session: err = %v, want ErrSessionNotFound", err)
	}
	if _, err := repo.FindValid(ctx, hash("still-in"), testNow); err != nil {
		t.Errorf("other session should remain: %v", err)
	}
	if err := repo.Delete(ctx, hash("never-existed")); err != nil {
		t.Errorf("deleting an unknown session: %v, want no error", err)
	}
}

func TestPostgresSessionRepository_DeleteExpired(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	mustCreate(t, repo, session("expired", testNow.Add(-time.Second)))
	mustCreate(t, repo, session("expiring-now", testNow))
	mustCreate(t, repo, session("valid", testNow.Add(time.Second)))

	n, err := repo.DeleteExpired(ctx, testNow)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if n != 2 {
		t.Errorf("deleted %d sessions, want 2", n)
	}
	if _, err := repo.FindValid(ctx, hash("valid"), testNow); err != nil {
		t.Errorf("unexpired session should remain: %v", err)
	}
	// Gone from the table, not just filtered out: they don't come back at an earlier time
	if _, err := repo.FindValid(ctx, hash("expired"), testNow.Add(-time.Hour)); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Errorf("expired session still stored: err = %v", err)
	}
}

func TestPostgresSessionRepository_Create_Rejects(t *testing.T) {
	repo := newRepo(t)
	mustCreate(t, repo, session("token", testNow.Add(time.Hour)))

	t.Run("a duplicate token", func(t *testing.T) {
		if err := repo.Create(context.Background(), session("token", testNow.Add(time.Hour))); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("a hash that isn't SHA-256 sized", func(t *testing.T) {
		s := session("token", testNow.Add(time.Hour))
		s.TokenHash = []byte("raw-token-stored-by-mistake")
		if err := repo.Create(context.Background(), s); err == nil {
			t.Fatal("expected an error")
		}
	})
}
