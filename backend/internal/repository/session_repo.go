package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionRepository interface {
	Create(ctx context.Context, session model.Session) error
	// FindValid returns the session with this token hash if it expires after now, or else
	// ErrSessionNotFound.
	FindValid(ctx context.Context, tokenHash []byte, now time.Time) (*model.Session, error)
	// Delete removes the session with this token hash. Deleting one that doesn't exist is not
	// an error.
	Delete(ctx context.Context, tokenHash []byte) error
	// DeleteExpired removes every session that expired at or before now and returns how many
	// it removed.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// PostgresSessionRepository stores sessions in the sessions table.
type PostgresSessionRepository struct {
	db *pgxpool.Pool
}

func NewPostgresSessionRepository(db *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, s model.Session) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_email, created_at, expires_at) VALUES ($1, $2, $3, $4)`,
		s.TokenHash, s.UserEmail, s.CreatedAt, s.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) FindValid(ctx context.Context, tokenHash []byte, now time.Time) (*model.Session, error) {
	var s model.Session
	err := r.db.QueryRow(ctx,
		`SELECT token_hash, user_email, created_at, expires_at FROM sessions WHERE token_hash = $1 AND expires_at > $2`,
		tokenHash, now).Scan(&s.TokenHash, &s.UserEmail, &s.CreatedAt, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select session: %w", err)
	}
	return &s, nil
}

func (r *PostgresSessionRepository) Delete(ctx context.Context, tokenHash []byte) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= $1`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
