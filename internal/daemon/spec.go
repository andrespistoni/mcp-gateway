package daemon

import (
	"fmt"
	"path/filepath"
	"strings"

	"mcp-gateway/internal/endpoint"
)

// Spec is deliberately limited to the binary and the validated listen port.
// It cannot carry downstream environment, project paths, sessions, or secrets.
type Spec struct {
	binary string
	port   endpoint.Port
}

func NewSpec(binary string, port endpoint.Port) (Spec, error) {
	if binary == "" || strings.ContainsRune(binary, 0) || !filepath.IsAbs(binary) {
		return Spec{}, fmt.Errorf("la ruta del binario debe ser absoluta")
	}
	if _, err := endpoint.NewPort(port.Number()); err != nil {
		return Spec{}, err
	}
	return Spec{binary: filepath.Clean(binary), port: port}, nil
}

func (s Spec) Valid() error {
	_, err := NewSpec(s.binary, s.port)
	return err
}

func (s Spec) Binary() string      { return s.binary }
func (s Spec) Port() endpoint.Port { return s.port }

// Args returns logical argv, never a shell command string.
func (s Spec) Args() []string { return []string{"serve", "--port", s.port.Decimal()} }
