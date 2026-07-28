package sse

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type SessionID [32]byte

func NewSessionID(entropy io.Reader) (SessionID, error) {
	if entropy == nil {
		entropy = rand.Reader
	}
	var id SessionID
	if _, err := io.ReadFull(entropy, id[:]); err != nil {
		return SessionID{}, fmt.Errorf("generar identificador de sesión: %w", err)
	}
	return id, nil
}

func ParseSessionID(value string) (SessionID, error) {
	if len(value) != 43 {
		return SessionID{}, fmt.Errorf("sessionId inválido")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return SessionID{}, fmt.Errorf("sessionId inválido")
	}
	var id SessionID
	copy(id[:], decoded)
	if id.Encode() != value {
		return SessionID{}, fmt.Errorf("sessionId inválido")
	}
	return id, nil
}

func (id SessionID) Encode() string { return base64.RawURLEncoding.EncodeToString(id[:]) }
