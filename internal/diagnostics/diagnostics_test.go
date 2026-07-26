package diagnostics

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestFaultPreservesCauseWithoutExposingIt(t *testing.T) {
	cause := errors.New("credencial cruda")
	fault := NewFault(Configuration, "configuración inválida", cause)
	if !errors.Is(fault, cause) {
		t.Fatal("Fault no conserva errors.Is")
	}
	var target *Fault
	if !errors.As(fault, &target) || target.Kind() != Configuration {
		t.Fatalf("errors.As/Kind = %#v", target)
	}
	if strings.Contains(fault.Error(), cause.Error()) || ExternalMessage(cause) != "error interno" {
		t.Fatal("se expuso una causa cruda")
	}
}

func TestSafeFieldRejectsSensitiveKeys(t *testing.T) {
	for _, key := range []string{"api_token", "ClientSecret", "PASSWORD", "privateKey", "authorization"} {
		if _, err := SafeField(key, "valor"); err == nil {
			t.Fatalf("SafeField(%q) debía rechazar la clave", key)
		}
	}
	if _, err := SafeField("ruta", "/tmp/config"); err != nil {
		t.Fatalf("campo seguro rechazado: %v", err)
	}
}

func TestSinkRedactsBeforeWriting(t *testing.T) {
	var normal, failures bytes.Buffer
	sink := NewSink(&normal, &failures)
	path, _ := SafeField("ruta", "/tmp/API_SECRET=oculto")
	if err := sink.Status("estado AUTH_TOKEN=valor", path, SecretField("detalle", "otro-secreto")); err != nil {
		t.Fatal(err)
	}
	output := normal.String()
	for _, secret := range []string{"valor", "oculto", "otro-secreto"} {
		if strings.Contains(output, secret) {
			t.Fatalf("sink filtró %q en %q", secret, output)
		}
	}
	if count := strings.Count(output, Redacted); count != 3 {
		t.Fatalf("redacciones = %d en %q", count, output)
	}
	if failures.Len() != 0 {
		t.Fatalf("stderr inesperado: %q", failures.String())
	}
}
