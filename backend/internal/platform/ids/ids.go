// Package ids generates identifiers (ADR-018): UUIDv7 primary keys, never sequential integers.
package ids

import "github.com/google/uuid"

// New returns a time-ordered UUIDv7 string.
func New() string { return uuid.Must(uuid.NewV7()).String() }

// NewUUID returns a time-ordered UUIDv7.
func NewUUID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Valid reports whether s is a canonical UUID.
func Valid(s string) bool {
	if len(s) != 36 {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}
