// Package config loads the application config: from a Parameter Manager parameter on Google
// Cloud, or from a local YAML file in development. Both hold the same YAML.
package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultPath       = "config.local.yml"
	defaultPort       = "8080"
	defaultSessionTTL = 30 * 24 * time.Hour
	minSecretLength   = 32
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Frontend FrontendConfig `yaml:"frontend"`
	Database DatabaseConfig `yaml:"database"`
	Session  SessionConfig  `yaml:"session"`
	User     UserConfig     `yaml:"user"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

// FrontendConfig holds the shared key that proves a request came from our frontend.
type FrontendConfig struct {
	APIKey string `yaml:"api_key"`
}

// DatabaseConfig locates the Postgres database.
type DatabaseConfig struct {
	// URL is a Postgres connection string. The DATABASE_URL environment variable overrides it;
	// on Cloud Run that's the Neon connection string, from Secret Manager.
	URL string `yaml:"url"`
}

// SessionConfig controls the sessions started on login.
type SessionConfig struct {
	TTL time.Duration `yaml:"ttl"`

	// Deprecated: sessions are stored in the database and no longer signed, so this is
	// ignored. It's still accepted so configs that set it keep loading; once it's gone from
	// every config (including backend-config), delete this field.
	Secret string `yaml:"secret"`
}

// UserConfig is the single user allowed to log in.
type UserConfig struct {
	Email        string `yaml:"email"`
	PasswordHash string `yaml:"password_hash"`
}

// Path returns the config file location: the CONFIG_PATH environment variable,
// or config.local.yml in the working directory.
func Path() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return defaultPath
}

// LoadFromEnv loads the config from the Parameter Manager parameter named by the
// CONFIG_PARAMETER environment variable (set on Cloud Run), or else from the file found by Path.
func LoadFromEnv(ctx context.Context) (*Config, error) {
	if parameter := os.Getenv("CONFIG_PARAMETER"); parameter != "" {
		return LoadParameter(ctx, parameter)
	}
	return Load(Path())
}

// LoadParameter loads the newest version of a Parameter Manager parameter, given as
// projects/PROJECT/locations/global/parameters/NAME, with its secret references resolved.
func LoadParameter(ctx context.Context, parameter string) (*Config, error) {
	data, err := renderLatest(ctx, parameter)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return parse(data, parameter)
}

// Load reads the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return parse(data, path)
}

// parse decodes YAML config, applies defaults and validates it. source names it in errors.
// The PORT and DATABASE_URL environment variables (set by Cloud Run) override server.port and
// database.url.
func parse(data []byte, source string) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // fail on typos such as "pasword_hash"
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", source, err)
	}

	cfg.applyDefaults()
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}
	if url := os.Getenv("DATABASE_URL"); url != "" {
		cfg.Database.URL = url
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", source, err)
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Port == "" {
		c.Server.Port = defaultPort
	}
	if c.Session.TTL == 0 {
		c.Session.TTL = defaultSessionTTL
	}
}

func (c *Config) validate() error {
	var errs []error
	if len(c.Frontend.APIKey) < minSecretLength {
		errs = append(errs, fmt.Errorf("frontend.api_key must be at least %d characters", minSecretLength))
	}
	if c.Database.URL == "" {
		errs = append(errs, errors.New("database.url (or the DATABASE_URL environment variable) is required"))
	}
	if c.Session.TTL < 0 {
		errs = append(errs, errors.New("session.ttl must be positive"))
	}
	if c.User.Email == "" {
		errs = append(errs, errors.New("user.email is required"))
	}
	if _, err := bcrypt.Cost([]byte(c.User.PasswordHash)); err != nil {
		errs = append(errs, errors.New("user.password_hash must be a bcrypt hash (generate one with: go run ./cmd/hashpassword)"))
	}
	return errors.Join(errs...)
}
