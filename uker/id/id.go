package id

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

// New returns a random 128-bit identifier encoded as hex.
func New() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// MustNew returns a new identifier panicking if generation fails.
func MustNew() string {
	id, err := New()
	if err != nil {
		panic(err)
	}
	return id
}

// Short returns a URL safe short identifier.
func Short() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
