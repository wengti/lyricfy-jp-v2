package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
)

const (
	testEmail    = "me@example.com"
	testPassword = "correct-password"
	testSecret   = "test-secret-that-is-at-least-32-characters"
	testTTL      = time.Hour
)

var testNow = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

// newTestService returns a service with a fixed clock that tests can move forward.
func newTestService(t *testing.T, secret string) (*Service, *time.Time) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	now := testNow
	svc := NewService(repository.NewConfigUserRepository(testEmail, string(hash)), secret, testTTL)
	svc.now = func() time.Time { return now }
	return svc, &now
}

func mustLogin(t *testing.T, svc *Service) *Session {
	t.Helper()
	session, err := svc.Login(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	return session
}

func TestLogin_Success(t *testing.T) {
	svc, _ := newTestService(t, testSecret)

	session := mustLogin(t, svc)

	if want := testNow.Add(testTTL); !session.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", session.ExpiresAt, want)
	}
	email, err := svc.VerifyToken(session.Token)
	if err != nil || email != testEmail {
		t.Fatalf("VerifyToken = %q, %v; want %q", email, err, testEmail)
	}
}

func TestLogin_RejectsBadCredentials(t *testing.T) {
	svc, _ := newTestService(t, testSecret)

	tests := []struct {
		name, email, password string
	}{
		{"wrong password", testEmail, "wrong-password"},
		{"unknown email", "someone@example.com", testPassword},
		{"empty password", testEmail, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("err = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestVerifyToken_ExpiredToken(t *testing.T) {
	svc, now := newTestService(t, testSecret)
	session := mustLogin(t, svc)

	*now = now.Add(testTTL + time.Second)

	if _, err := svc.VerifyToken(session.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_RejectsForgedTokens(t *testing.T) {
	svc, _ := newTestService(t, testSecret)
	session := mustLogin(t, svc)

	otherSvc, _ := newTestService(t, "a-completely-different-secret-of-32-chars")
	otherSession := mustLogin(t, otherSvc)

	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Issuer:    tokenIssuer,
		Subject:   testEmail,
		ExpiresAt: jwt.NewNumericDate(testNow.Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	noExpiry, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:  tokenIssuer,
		Subject: testEmail,
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}

	// A genuine token with its payload edited to claim another user; the signature no longer matches
	parts := strings.Split(session.Token, ".")
	forgedClaims := fmt.Sprintf(`{"iss":%q,"sub":"attacker@example.com","exp":%d}`, tokenIssuer, testNow.Add(time.Hour).Unix())
	tamperedPayload := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(forgedClaims)) + "." + parts[2]

	tests := []struct {
		name, token string
	}{
		{"signed with a different secret", otherSession.Token},
		{"alg none", noneToken},
		{"no expiry", noExpiry},
		{"tampered payload", tamperedPayload},
		{"not a token", "garbage"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.VerifyToken(tt.token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}
