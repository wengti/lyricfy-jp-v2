package repository

import (
	"context"
	"errors"
	"testing"
)

func TestConfigUserRepository_FindByEmail(t *testing.T) {
	repo := NewConfigUserRepository("Me@Example.com", "hash")

	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"exact match", "Me@Example.com", nil},
		{"different case", "me@example.com", nil},
		{"surrounding whitespace", "  me@example.com ", nil},
		{"different email", "someone@example.com", ErrUserNotFound},
		{"empty", "", ErrUserNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.FindByEmail(context.Background(), tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && user.PasswordHash != "hash" {
				t.Errorf("unexpected user %+v", user)
			}
		})
	}
}
