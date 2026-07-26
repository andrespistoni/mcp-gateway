package endpoint

import (
	"fmt"
	"strconv"
)

const (
	MinPort     = 1024
	MaxPort     = 65535
	DefaultPort = 3333
)

// Port solo puede representar un puerto permitido por el gateway.
type Port struct {
	number uint16
}

func NewPort(number int) (Port, error) {
	if number < MinPort || number > MaxPort {
		return Port{}, fmt.Errorf("el puerto debe estar entre %d y %d", MinPort, MaxPort)
	}
	return Port{number: uint16(number)}, nil
}

func ParsePort(value string) (Port, error) {
	if value == "" {
		return Port{}, fmt.Errorf("el puerto está vacío")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return Port{}, fmt.Errorf("el puerto debe ser decimal")
		}
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return Port{}, fmt.Errorf("puerto inválido: %w", err)
	}
	return NewPort(number)
}

func MustPort(number int) Port {
	port, err := NewPort(number)
	if err != nil {
		panic(err)
	}
	return port
}

func (p Port) Number() int {
	return int(p.number)
}

func (p Port) Decimal() string {
	return strconv.Itoa(p.Number())
}

func (p Port) MarshalYAML() (any, error) {
	if _, err := NewPort(p.Number()); err != nil {
		return nil, err
	}
	return p.Number(), nil
}

// ResolvePort aplica la precedencia CLI, configuración y valor predeterminado.
func ResolvePort(flag, configured *Port) Port {
	if flag != nil {
		return *flag
	}
	if configured != nil {
		return *configured
	}
	return MustPort(DefaultPort)
}
