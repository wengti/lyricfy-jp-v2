// Package config loads the application config from a local YAML file.
package config

import (
	"bytes"
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

// SessionConfig controls the session tokens issued on login.
type SessionConfig struct {
	Secret string        `yaml:"secret"`
	TTL    time.Duration `yaml:"ttl"`
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

// LoadFromEnv loads the config file found by Path.
func LoadFromEnv() (*Config, error) {
	return Load(Path())
}

// Load reads the config file at path, applies defaults and validates it.
// The PORT environment variable (set by Cloud Run) overrides server.port.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // fail on typos such as "pasword_hash"
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg.applyDefaults()
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
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
	if len(c.Session.Secret) < minSecretLength {
		errs = append(errs, fmt.Errorf("session.secret must be at least %d characters", minSecretLength))
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
