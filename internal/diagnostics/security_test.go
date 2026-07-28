package diagnostics

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdversarialRedactionAcrossSinkChannels(t *testing.T) {
	secrets := []string{"tok-123", "sec-456", "pwd-789", "key-abc", "auth-def"}
	message := "TOKEN=tok-123 secret:sec-456 PASSWORD = pwd-789 api_key=key-abc oauthAUTH:auth-def \"AUTH_TOKEN\":\"tok-123\" 'privateKey': 'key-abc'"
	var normal, errors bytes.Buffer
	sink := NewSink(&normal, &errors)
	safe, err := SafeField("detail", "nested AUTH_TOKEN=tok-123")
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Status(message, safe, SecretField("token", "tok-123")); err != nil {
		t.Fatal(err)
	}
	if err := sink.Error(message, SecretField("password", "pwd-789")); err != nil {
		t.Fatal(err)
	}
	output := normal.String() + errors.String()
	for _, secret := range secrets {
		if strings.Contains(output, secret) {
			t.Fatalf("el sink expuso %q: %q", secret, output)
		}
	}
	if strings.Count(output, Redacted) < 8 {
		t.Fatalf("redacciones insuficientes: %q", output)
	}
}

func TestSafeFieldRejectsSensitiveNamesWithoutExceptions(t *testing.T) {
	for _, key := range []string{"token", "SECRET", "password_hint", "apiKey", "authorization"} {
		if _, err := SafeField(key, "valor"); err == nil {
			t.Fatalf("SafeField(%q) aceptó una clave sensible", key)
		}
	}
}
