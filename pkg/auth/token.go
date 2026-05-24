package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Type string

const (
	Read  Type = "read"
	Write Type = "write"
)

type Token struct {
	typ  Type
	raw  string
	hash string
}

// Generate creates a new token of the given type.
// Uses crypto/rand for 32 bytes of cryptographic randomness.
func Generate(t Type) Token {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("auth: failed to read random bytes: %v", err))
	}
	raw := hex.EncodeToString(b)
	digest := sha256.Sum256([]byte(raw))
	return Token{
		typ:  t,
		raw:  raw,
		hash: hex.EncodeToString(digest[:]),
	}
}

func FromRaw(raw string, t Type) Token {
	digest := sha256.Sum256([]byte(raw))
	return Token{
		typ:  t,
		raw:  raw,
		hash: hex.EncodeToString(digest[:]),
	}
}

// Exists only when generating
func (t Token) Raw() string {
	return t.raw
}

func (t Token) Hash() string {
	return t.hash
}

func (t Token) Type() Type {
	return t.typ
}
