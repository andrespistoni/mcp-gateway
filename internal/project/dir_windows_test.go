//go:build windows

package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWindowsAceptaRaizYDirectorio(t *testing.T) {
	directory := t.TempDir()
	if _, err := Resolve(directory); err != nil {
		t.Fatal(err)
	}
	volume := filepath.VolumeName(directory)
	if volume != "" {
		if _, err := Resolve(volume + `\`); err != nil {
			t.Fatalf("raíz de volumen: %v", err)
		}
	}
	if len(directory) >= 2 && directory[:2] == `\\` {
		if _, err := Resolve(directory); err != nil {
			t.Fatalf("UNC nativa: %v", err)
		}
	}
}

func TestResolveWindowsSymlinkYRechazos(t *testing.T) {
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(directory, alias); err == nil {
		original, _ := Resolve(directory)
		linked, err := Resolve(alias)
		if err != nil || linked.Path() != alias || !Compare(original, linked) {
			t.Fatalf("symlink = %#v, %v", linked, err)
		}
	}
	for _, path := range []string{"relative", `\rooted-but-no-volume`} {
		if _, err := Resolve(path); err == nil {
			t.Errorf("Resolve(%q) debía fallar", path)
		}
	}
}
