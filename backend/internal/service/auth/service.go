// Package auth holds the authentication business logic: checking credentials
// and issuing and verifying session tokens.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

const tokenIssuer = "lyricfy-api"

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
	users  repository.UserRepository
	secret []byte
	ttl    time.Duration
	now    func() time.Time // replaceable in tests
}

func NewService(users repository.UserRepository, secret string, ttl time.Duration) *Service {
	return &Service{users: users, secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Login checks the credentials and returns a signed session token.
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

	now := s.now()
	expiresAt := now.Add(s.ttl)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    tokenIssuer,
		Subject:   user.Email,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	})
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}
	return &Session{Token: signed, ExpiresAt: expiresAt}, nil
}

// VerifyToken checks a session token's signature and expiry and returns the user's email.
func (s *Service) VerifyToken(tokenString string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // rejects "none" and other algorithms
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
