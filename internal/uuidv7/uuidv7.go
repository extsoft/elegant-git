// Package uuidv7 generates time-ordered UUID version 7 identifiers.
package uuidv7

import "github.com/google/uuid"

// New returns a new UUIDv7 string.
func New() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
