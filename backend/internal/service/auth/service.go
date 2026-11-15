// Package auth holds the authentication business logic: checking credentials and starting,
// verifying and ending sessions.
//
// A session token is 32 random bytes, base64url-encoded, that the browser keeps in a cookie.
// The database stores only its SHA-256 hash with an expiry, so logging out deletes the row and
// the token stops working at once. See docs/session-tokens.md.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

// tokenBytes of randomness make a token impossible to guess (256 bits), which is also why a
// fast hash is safe to store it under, unlike a password
const tokenBytes = 32

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired session")
)

// dummyHash is compared against when the email is unknown, so a wrong email takes as long
// as a wrong password and response times don't reveal which one was wrong.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Service struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	ttl      time.Duration
	now      func() time.Time // replaceable in tests
}

func NewService(users repository.UserRepository, sessions repository.SessionRepository, ttl time.Duration) *Service {
	return &Service{users: users, sessions: sessions, ttl: ttl, now: time.Now}
}

// Login checks the credentials and starts a session that lasts the configured TTL. It also
// deletes expired sessions: they're already rejected, so this only keeps the table small.
func (s *Service) Login(ctx context.Context, email, password string) (*Session, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		// even if a user is not found, still compare with a dummy password to ensure same processing time
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Whole seconds: Postgres keeps microseconds, so the expiry sent back matches the stored one
	now := s.now().UTC().Truncate(time.Second)
	if _, err := s.sessions.DeleteExpired(ctx, now); err != nil {
		return nil, fmt.Errorf("delete expired sessions: %w", err)
	}

	token := newToken()
	expiresAt := now.Add(s.ttl)
	err = s.sessions.Create(ctx, model.Session{
		TokenHash: hashToken(token),
		UserEmail: user.Email,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &Session{Token: token, ExpiresAt: expiresAt}, nil
}

// VerifyToken returns the email of the user a session token belongs to. It returns
// ErrInvalidToken if the token is malformed, unknown, expired or logged out, and any other
// error if the check itself failed (e.g. the database is unreachable).
func (s *Service) VerifyToken(ctx context.Context, token string) (string, error) {
	if !wellFormed(token) {
		return "", ErrInvalidToken // can't be one of ours; skip the database
	}
	session, err := s.sessions.FindValid(ctx, hashToken(token), s.now())
	if errors.Is(err, repository.ErrSessionNotFound) {
		return "", ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("find session: %w", err)
	}
	return session.UserEmail, nil
}

// Logout ends a session, so its token stops working even before it expires. Ending a session
// that is unknown, expired or already ended is not an error.
func (s *Service) Logout(ctx context.Context, token string) error {
	if !wellFormed(token) {
		return nil
	}
	if err := s.sessions.Delete(ctx, hashToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// newToken returns a new random session token.
func newToken() string {
	b := make([]byte, tokenBytes)
	rand.Read(b) // never returns an error; crashes the program if randomness is unavailable
	return base64.RawURLEncoding.EncodeToString(b)
}

// wellFormed reports whether token has the shape newToken produces.
func wellFormed(token string) bool {
	b, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(b) == tokenBytes
}

// hashToken returns what the database stores for a token.
func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
