//go:build !windows

package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAceptaRutaLimpiaRaizYSymlink(t *testing.T) {
	directory := t.TempDir()
	dirty := filepath.Join(directory, ".", "child", "..")
	resolved, err := Resolve(dirty)
	if err != nil || resolved.Path() != directory {
		t.Fatalf("Resolve(%q) = %q, %v", dirty, resolved.Path(), err)
	}
	root, err := Resolve(string(filepath.Separator))
	if err != nil || root.Path() != string(filepath.Separator) {
		t.Fatalf("Resolve(root) = %q, %v", root.Path(), err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Fatal(err)
	}
	aliasDir, err := Resolve(alias)
	if err != nil {
		t.Fatal(err)
	}
	if aliasDir.Path() != alias || !Compare(resolved, aliasDir) {
		t.Fatalf("el alias no se preservó/comparó: %#v %#v", resolved, aliasDir)
	}
}

func TestResolveRechazaEntradasInvalidas(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative", filepath.Join(t.TempDir(), "missing"), file, `\\server\share`} {
		if _, err := Resolve(path); err == nil {
			t.Errorf("Resolve(%q) debía fallar", path)
		}
	}
}

func TestFromCurrent(t *testing.T) {
	current, err := FromCurrent()
	if err != nil || !filepath.IsAbs(current.Path()) {
		t.Fatalf("FromCurrent = %q, %v", current.Path(), err)
	}
}
