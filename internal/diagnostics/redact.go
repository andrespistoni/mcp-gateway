package diagnostics

import (
	"regexp"
	"strings"
)

const Redacted = "[REDACTADO]"

var sensitiveFragments = [...]string{"TOKEN", "SECRET", "PASSWORD", "KEY", "AUTH"}

var structuredSecret = regexp.MustCompile(`(?i)([A-Za-z_][A-Za-z0-9_.-]*(?:TOKEN|SECRET|PASSWORD|KEY|AUTH)[A-Za-z0-9_.-]*)(\s*[:=]\s*)("[^"]*"|[^\s,;}]+)`)

func IsSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, fragment := range sensitiveFragments {
		if strings.Contains(upper, fragment) {
			return true
		}
	}
	return false
}

func RedactValue(key, value string) string {
	if IsSensitiveKey(key) {
		return Redacted
	}
	return value
}

// RedactText cubre formas estructuradas simples KEY=VALUE y KEY: VALUE.
func RedactText(value string) string {
	return structuredSecret.ReplaceAllString(value, `${1}${2}`+Redacted)
}
