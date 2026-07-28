package diagnostics

import (
	"strings"
	"testing"
)

func TestRedactTextRedactsExactSensitiveKey(t *testing.T) {
	got := RedactText("TOKEN=private API_SECRET: also-private")
	if strings.Contains(got, "private") || strings.Contains(got, "also-private") || !strings.Contains(got, Redacted) {
		t.Fatalf("redaction=%q", got)
	}
}
