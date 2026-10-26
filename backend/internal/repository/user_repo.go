// Package repository is the data access layer.
package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
}

// ConfigUserRepository serves the single user defined in the config file.
// It will be replaced by a Postgres-backed repository once the database exists.
type ConfigUserRepository struct {
	user model.User
}

func NewConfigUserRepository(email, passwordHash string) *ConfigUserRepository {
	return &ConfigUserRepository{user: model.User{Email: email, PasswordHash: passwordHash}}
}

// FindByEmail matches emails case-insensitively, ignoring surrounding whitespace.
func (r *ConfigUserRepository) FindByEmail(_ context.Context, email string) (*model.User, error) {
	if !strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(r.user.Email)) {
		return nil, ErrUserNotFound
	}
	u := r.user
	return &u, nil
}
