package project

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestFromHTTPPrecedenciaFallbackYAusencia(t *testing.T) {
	directory := t.TempDir()
	encoded := url.QueryEscape(directory)
	value, err := FromHTTP(&encoded, nil)
	if err != nil || !value.Present() || value.Path() != directory {
		t.Fatalf("query = %#v, %v", value, err)
	}
	headerValue := directory
	value, err = FromHTTP(nil, &headerValue)
	if err != nil || value.Path() != directory {
		t.Fatalf("header = %#v, %v", value, err)
	}
	value, err = FromHTTP(nil, nil)
	if err != nil || value.Present() || value.Path() != "" {
		t.Fatalf("ausente = %#v, %v", value, err)
	}
}

func TestFromHTTPComparaAliasYConservaQuery(t *testing.T) {
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Skipf("symlink no disponible: %v", err)
	}
	encoded := url.QueryEscape(alias)
	header := directory
	value, err := FromHTTP(&encoded, &header)
	if err != nil || value.Path() != alias {
		t.Fatalf("FromHTTP = %q, %v", value.Path(), err)
	}
}

func TestFromHTTPRechazaCodificacionConflictoYRutasInvalidas(t *testing.T) {
	directory := t.TempDir()
	other := t.TempDir()
	invalidUTF8 := "%ff"
	badEscape := "%zz"
	relative := "relative"
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		query  *string
		header *string
	}{
		{query: &badEscape}, {query: &invalidUTF8}, {query: &relative},
		{query: stringPointer(url.QueryEscape(directory)), header: &other},
		{header: &file},
	}
	for _, test := range tests {
		if _, err := FromHTTP(test.query, test.header); err == nil {
			t.Errorf("FromHTTP(%v, %v) debía fallar", test.query, test.header)
		}
	}
}

func stringPointer(value string) *string { return &value }
