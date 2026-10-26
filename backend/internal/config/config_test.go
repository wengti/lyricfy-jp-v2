package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	validKey  = strings.Repeat("k", 32)
	validHash = mustHash("secret")
)

func mustHash(password string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validYAML() string {
	return `
server:
  port: "9000"
frontend:
  api_key: "` + validKey + `"
session:
  secret: "` + validKey + `"
  ttl: 24h
user:
  email: "me@example.com"
  password_hash: "` + validHash + `"
`
}

func TestPath(t *testing.T) {
	t.Run("defaults to config.local.yml", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "")
		if got := Path(); got != "config.local.yml" {
			t.Errorf("Path() = %q, want config.local.yml", got)
		}
	})
	t.Run("uses CONFIG_PATH when set", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "/etc/lyricfy/config.yml")
		if got := Path(); got != "/etc/lyricfy/config.yml" {
			t.Errorf("Path() = %q, want the CONFIG_PATH value", got)
		}
	})
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("CONFIG_PATH", writeConfig(t, validYAML()))
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.User.Email != "me@example.com" {
		t.Errorf("email = %q, want the file's value", cfg.User.Email)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	t.Setenv("PORT", "")
	cfg, err := Load(writeConfig(t, validYAML()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != "9000" {
		t.Errorf("port = %q, want 9000", cfg.Server.Port)
	}
	if cfg.Session.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", cfg.Session.TTL)
	}
	if cfg.User.Email != "me@example.com" {
		t.Errorf("email = %q", cfg.User.Email)
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	yaml := strings.NewReplacer(`  port: "9000"`, "", "  ttl: 24h", "").Replace(validYAML())

	cfg, err := Load(writeConfig(t, yaml))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != defaultPort {
		t.Errorf("port = %q, want default %q", cfg.Server.Port, defaultPort)
	}
	if cfg.Session.TTL != defaultSessionTTL {
		t.Errorf("ttl = %v, want default %v", cfg.Session.TTL, defaultSessionTTL)
	}
}

func TestLoad_PortEnvOverridesFile(t *testing.T) {
	t.Setenv("PORT", "1234")
	cfg, err := Load(writeConfig(t, validYAML()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != "1234" {
		t.Errorf("port = %q, want 1234 from PORT env", cfg.Server.Port)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yml")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoad_RejectsUnknownFields(t *testing.T) {
	yaml := strings.Replace(validYAML(), "password_hash:", "pasword_hash:", 1)
	_, err := Load(writeConfig(t, yaml))
	if err == nil || !strings.Contains(err.Error(), "pasword_hash") {
		t.Fatalf("expected an error naming the misspelt field, got %v", err)
	}
}

func TestLoad_Validation(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr string
	}{
		{"short frontend key", `api_key: "` + validKey + `"`, `api_key: "short"`, "frontend.api_key"},
		{"short session secret", `secret: "` + validKey + `"`, `secret: "short"`, "session.secret"},
		{"missing email", `email: "me@example.com"`, `email: ""`, "user.email"},
		{"plaintext password", `password_hash: "` + validHash + `"`, `password_hash: "secret"`, "user.password_hash"},
		{"negative ttl", "ttl: 24h", "ttl: -1h", "session.ttl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := strings.Replace(validYAML(), tt.from, tt.to, 1)
			_, err := Load(writeConfig(t, yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected an error mentioning %q, got %v", tt.wantErr, err)
			}
		})
	}
}
