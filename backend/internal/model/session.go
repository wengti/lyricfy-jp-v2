package model

import "time"

// Session is one login. The browser holds a random token; TokenHash is its SHA-256, and the
// token itself is never stored.
type Session struct {
	TokenHash []byte
	UserEmail string
	CreatedAt time.Time
	ExpiresAt time.Time
}
