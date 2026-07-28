package sse

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
)

type cursorCodec struct {
	identity [32]byte
	key      [32]byte
}

func newCursorCodec(identity [32]byte, entropy io.Reader) (cursorCodec, error) {
	var codec cursorCodec
	codec.identity = identity
	if _, err := io.ReadFull(entropy, codec.key[:]); err != nil {
		return cursorCodec{}, fmt.Errorf("generar clave de cursor: %w", err)
	}
	return codec, nil
}

func (c cursorCodec) encode(position int) string {
	payload := make([]byte, 36)
	copy(payload, c.identity[:])
	binary.BigEndian.PutUint32(payload[32:], uint32(position))
	mac := hmac.New(sha256.New, c.key[:])
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(append(payload[32:], mac.Sum(nil)[:16]...))
}

func (c cursorCodec) decode(value string, maximum int) (int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != 20 || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return 0, fmt.Errorf("cursor inválido")
	}
	payload := make([]byte, 36)
	copy(payload, c.identity[:])
	copy(payload[32:], decoded[:4])
	mac := hmac.New(sha256.New, c.key[:])
	_, _ = mac.Write(payload)
	if !hmac.Equal(decoded[4:], mac.Sum(nil)[:16]) {
		return 0, fmt.Errorf("cursor inválido")
	}
	position := int(binary.BigEndian.Uint32(decoded[:4]))
	if position <= 0 || position >= maximum {
		return 0, fmt.Errorf("cursor inválido")
	}
	return position, nil
}
