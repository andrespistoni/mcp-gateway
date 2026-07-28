package diagnostics

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestChecksAndFaultAccessors(t *testing.T) {
	pass := Passed("config", "válida")
	failCause := errors.New("fallo")
	fail := Failed("daemon", "detenido", failCause)
	if !pass.OK() || fail.OK() || fail.Err != failCause {
		t.Fatalf("checks = %#v, %#v", pass, fail)
	}

	field, err := SafeField("ruta", "/tmp/demo")
	if err != nil {
		t.Fatal(err)
	}
	fault := NewFault(Configuration, "inválida", failCause, field)
	if fault.Message() != "inválida" || KindOf(fault) != Configuration || KindOf(failCause) != "" {
		t.Fatalf("fault accessors = %q, %q", fault.Message(), KindOf(fault))
	}
	fields := fault.Fields()
	if len(fields) != 1 || fields[0].Key() != "ruta" || fields[0].RedactedValue() != "/tmp/demo" {
		t.Fatalf("fields = %#v", fields)
	}
	fields[0] = SecretField("otro", "valor")
	if fault.Fields()[0].Key() != "ruta" {
		t.Fatal("Fields no devolvió una copia")
	}
	if ExternalMessage(fault) != "inválida" {
		t.Fatalf("ExternalMessage = %q", ExternalMessage(fault))
	}
}

func TestRedactionAndSinkErrorChannel(t *testing.T) {
	if RedactValue("api_token", "secret") != Redacted || RedactValue("name", "visible") != "visible" {
		t.Fatal("RedactValue no aplicó sensibilidad")
	}
	if value := RedactText(`{"api_token":"secret","name":"visible"}`); strings.Contains(value, "secret") || !strings.Contains(value, "visible") {
		t.Fatalf("RedactText = %q", value)
	}
	secret := SecretField("token", "raw")
	if secret.Key() != "token" || secret.RedactedValue() != Redacted {
		t.Fatalf("SecretField = %#v", secret)
	}

	var normal, failures bytes.Buffer
	sink := NewSink(&normal, &failures)
	if err := sink.Error("falló", secret); err != nil {
		t.Fatal(err)
	}
	if normal.Len() != 0 || !strings.Contains(failures.String(), "falló") || strings.Contains(failures.String(), "raw") {
		t.Fatalf("sink outputs = %q / %q", normal.String(), failures.String())
	}
}
