package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/model"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository/repotest"
)

const (
	testEmail    = "me@example.com"
	testPassword = "correct-password"
	testTTL      = time.Hour
)

var testNow = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

// newTestService returns a service backed by an in-memory session store, with a fixed clock
// that tests can move forward.
func newTestService(t *testing.T) (*Service, *repotest.SessionRepository, *time.Time) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	sessions := repotest.NewSessionRepository()
	now := testNow
	svc := NewService(repository.NewConfigUserRepository(testEmail, string(hash)), sessions, testTTL)
	svc.now = func() time.Time { return now }
	return svc, sessions, &now
}

func mustLogin(t *testing.T, svc *Service) *Session {
	t.Helper()
	session, err := svc.Login(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	return session
}

func sha(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func TestLogin_Success(t *testing.T) {
	svc, _, _ := newTestService(t)

	session := mustLogin(t, svc)

	if want := testNow.Add(testTTL); !session.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", session.ExpiresAt, want)
	}
	email, err := svc.VerifyToken(context.Background(), session.Token)
	if err != nil || email != testEmail {
		t.Fatalf("VerifyToken = %q, %v; want %q", email, err, testEmail)
	}
}

func TestLogin_StoresOnlyTheHash(t *testing.T) {
	svc, sessions, _ := newTestService(t)

	session := mustLogin(t, svc)

	stored := sessions.All()
	if len(stored) != 1 {
		t.Fatalf("stored %d sessions, want 1", len(stored))
	}
	want := model.Session{TokenHash: sha(session.Token), UserEmail: testEmail, CreatedAt: testNow, ExpiresAt: testNow.Add(testTTL)}
	got := stored[0]
	if !bytes.Equal(got.TokenHash, want.TokenHash) || got.UserEmail != want.UserEmail ||
		!got.CreatedAt.Equal(want.CreatedAt) || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("stored %+v, want %+v", got, want)
	}
	if bytes.Contains(got.TokenHash, []byte(session.Token)) {
		t.Error("the raw token must never be stored")
	}
}

func TestLogin_IssuesRandomTokens(t *testing.T) {
	svc, _, _ := newTestService(t)

	a, b := mustLogin(t, svc), mustLogin(t, svc)

	if a.Token == b.Token {
		t.Fatal("two logins got the same token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(a.Token)
	if err != nil || len(raw) != 32 {
		t.Errorf("token %q should be 32 base64url-encoded bytes (err %v, %d bytes)", a.Token, err, len(raw))
	}
	// Both sessions stay valid: logging in again doesn't log out another device
	for _, s := range []*Session{a, b} {
		if _, err := svc.VerifyToken(context.Background(), s.Token); err != nil {
			t.Errorf("VerifyToken: %v", err)
		}
	}
}

func TestLogin_DeletesExpiredSessions(t *testing.T) {
	svc, sessions, now := newTestService(t)
	old := mustLogin(t, svc)

	*now = now.Add(testTTL + time.Second) // the first session has expired
	current := mustLogin(t, svc)

	stored := sessions.All()
	if len(stored) != 1 || !bytes.Equal(stored[0].TokenHash, sha(current.Token)) {
		t.Fatalf("want only the new session stored, got %d sessions", len(stored))
	}
	if _, err := svc.VerifyToken(context.Background(), old.Token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("old token: err = %v, want ErrInvalidToken", err)
	}
}

func TestLogin_KeepsUnexpiredSessions(t *testing.T) {
	svc, sessions, now := newTestService(t)
	mustLogin(t, svc)

	*now = now.Add(testTTL - time.Second) // the first session still has a second left
	mustLogin(t, svc)

	if n := len(sessions.All()); n != 2 {
		t.Errorf("stored %d sessions, want 2", n)
	}
}

func TestLogin_RejectsBadCredentials(t *testing.T) {
	svc, sessions, _ := newTestService(t)

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
	if n := len(sessions.All()); n != 0 {
		t.Errorf("failed logins stored %d sessions, want 0", n)
	}
}

func TestLogin_DatabaseError(t *testing.T) {
	svc, sessions, _ := newTestService(t)
	sessions.Err = errors.New("connection refused")

	_, err := svc.Login(context.Background(), testEmail, testPassword)

	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want the database error", err)
	}
}

func TestVerifyToken_ExpiredToken(t *testing.T) {
	svc, _, now := newTestService(t)
	session := mustLogin(t, svc)

	*now = now.Add(testTTL)

	if _, err := svc.VerifyToken(context.Background(), session.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_RejectsUnknownTokens(t *testing.T) {
	svc, _, _ := newTestService(t)
	session := mustLogin(t, svc)

	tests := []struct {
		name, token string
	}{
		{"well formed but never issued", base64.RawURLEncoding.EncodeToString(make([]byte, 32))},
		{"the stored hash instead of the token", base64.RawURLEncoding.EncodeToString(sha(session.Token))},
		{"issued token with a character changed", flipLastChar(session.Token)},
		{"too short", session.Token[:20]},
		{"padded base64", session.Token + "="},
		{"a JWT", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJtZUBleGFtcGxlLmNvbSJ9.c2lnbmF0dXJl"},
		{"not base64", strings.Repeat("!", 43)},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.VerifyToken(context.Background(), tt.token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func flipLastChar(token string) string {
	last := token[len(token)-1]
	replacement := byte('A')
	if last == 'A' {
		replacement = 'B'
	}
	return token[:len(token)-1] + string(replacement)
}

func TestVerifyToken_DatabaseErrorIsNotAnInvalidToken(t *testing.T) {
	svc, sessions, _ := newTestService(t)
	session := mustLogin(t, svc)
	sessions.Err = errors.New("connection refused")

	_, err := svc.VerifyToken(context.Background(), session.Token)

	// Callers must be able to tell an outage from a bad token, so an outage doesn't log you out
	if err == nil || errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want the database error", err)
	}
}

func TestLogout_EndsTheSessionBeforeItExpires(t *testing.T) {
	svc, _, _ := newTestService(t)
	session := mustLogin(t, svc)
	other := mustLogin(t, svc)

	if err := svc.Logout(context.Background(), session.Token); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if _, err := svc.VerifyToken(context.Background(), session.Token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("logged-out token: err = %v, want ErrInvalidToken", err)
	}
	if _, err := svc.VerifyToken(context.Background(), other.Token); err != nil {
		t.Errorf("another session should stay valid: %v", err)
	}
}

func TestLogout_UnknownTokensSucceed(t *testing.T) {
	svc, _, now := newTestService(t)
	session := mustLogin(t, svc)
	if err := svc.Logout(context.Background(), session.Token); err != nil {
		t.Fatal(err)
	}
	expired := mustLogin(t, svc)
	*now = now.Add(2 * testTTL)

	for name, token := range map[string]string{
		"already logged out": session.Token,
		"expired":            expired.Token,
		"malformed":          "garbage",
		"empty":              "",
	} {
		t.Run(name, func(t *testing.T) {
			if err := svc.Logout(context.Background(), token); err != nil {
				t.Errorf("Logout: %v, want no error", err)
			}
		})
	}
}

func TestLogout_DatabaseError(t *testing.T) {
	svc, sessions, _ := newTestService(t)
	session := mustLogin(t, svc)
	sessions.Err = errors.New("connection refused")

	if err := svc.Logout(context.Background(), session.Token); err == nil {
		t.Fatal("expected the database error")
	}
}
