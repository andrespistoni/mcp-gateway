package diagnostics

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type Field struct {
	key    string
	value  string
	secret bool
}

// SafeField rechaza claves sensibles para impedir etiquetar secretos como seguros.
func SafeField(key, value string) (Field, error) {
	if strings.TrimSpace(key) == "" {
		return Field{}, fmt.Errorf("la clave diagnóstica está vacía")
	}
	if IsSensitiveKey(key) {
		return Field{}, fmt.Errorf("la clave diagnóstica es sensible")
	}
	return Field{key: key, value: value}, nil
}

func SecretField(key, value string) Field {
	return Field{key: key, value: value, secret: true}
}

func (f Field) Key() string {
	return f.key
}

func (f Field) RedactedValue() string {
	if f.secret || IsSensitiveKey(f.key) {
		return Redacted
	}
	return RedactText(f.value)
}

type Sink struct {
	normal io.Writer
	error  io.Writer
	mu     sync.Mutex
}

func NewSink(normal, errorOutput io.Writer) *Sink {
	return &Sink{normal: normal, error: errorOutput}
}

func (s *Sink) Status(message string, fields ...Field) error {
	return s.write(s.normal, message, fields)
}

func (s *Sink) Error(message string, fields ...Field) error {
	return s.write(s.error, message, fields)
}

func (s *Sink) write(output io.Writer, message string, fields []Field) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var line strings.Builder
	line.WriteString(RedactText(message))
	for _, field := range fields {
		line.WriteByte(' ')
		line.WriteString(field.Key())
		line.WriteByte('=')
		line.WriteString(field.RedactedValue())
	}
	line.WriteByte('\n')
	_, err := io.WriteString(output, line.String())
	return err
}
