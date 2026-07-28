package proxy

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsUnavailable(t *testing.T) {
	if IsUnavailable(nil) {
		t.Fatal("nil no debe clasificarse como downstream no disponible")
	}
	if !IsUnavailable(fmt.Errorf("inicio fallido: %w", ErrDownstreamUnavailable)) {
		t.Fatal("el error envuelto debe conservar la clasificación")
	}
	if IsUnavailable(errors.New("otro error")) {
		t.Fatal("un error ajeno no debe clasificarse como downstream no disponible")
	}
}
