// Package repotest has in-memory repositories for tests that don't need a real database. They
// follow the same contracts as the Postgres ones, whose own tests run against a real database.
package repotest

import (
	"context"
	"sync"
	"time"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

// SessionRepository is an in-memory repository.SessionRepository.
type SessionRepository struct {
	// Err, when set, is returned by every method, to test what happens when the database fails
	Err error

	mu       sync.Mutex
	sessions map[string]model.Session // by token hash
}

var _ repository.SessionRepository = (*SessionRepository)(nil)

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{sessions: map[string]model.Session{}}
}

func (r *SessionRepository) Create(_ context.Context, s model.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	r.sessions[string(s.TokenHash)] = s
	return nil
}

func (r *SessionRepository) FindValid(_ context.Context, tokenHash []byte, now time.Time) (*model.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return nil, r.Err
	}
	s, ok := r.sessions[string(tokenHash)]
	if !ok || !s.ExpiresAt.After(now) {
		return nil, repository.ErrSessionNotFound
	}
	return &s, nil
}

func (r *SessionRepository) Delete(_ context.Context, tokenHash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	delete(r.sessions, string(tokenHash))
	return nil
}

func (r *SessionRepository) DeleteExpired(_ context.Context, now time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	var n int64
	for hash, s := range r.sessions {
		if !s.ExpiresAt.After(now) {
			delete(r.sessions, hash)
			n++
		}
	}
	return n, nil
}

// All returns every stored session, expired or not, so tests can check what was stored.
func (r *SessionRepository) All() []model.Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]model.Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		all = append(all, s)
	}
	return all
}
