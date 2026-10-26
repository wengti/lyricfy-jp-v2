// Package model holds the data structures shared across layers.
package model

type User struct {
	Email        string `json:"email"`
	PasswordHash string `json:"-"` // never serialised
}
