package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir conserva la forma absoluta limpia elegida por el caller y una ruta real
// opcional que se usa solo para comparar aliases.
type Dir struct {
	path string
	real string
}

func Resolve(path string) (Dir, error) {
	if path == "" || !filepath.IsAbs(path) {
		return Dir{}, fmt.Errorf("projectDir debe ser una ruta absoluta")
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return Dir{}, fmt.Errorf("consultar projectDir: %w", err)
	}
	if !info.IsDir() {
		return Dir{}, fmt.Errorf("projectDir no es un directorio")
	}
	real, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return Dir{}, fmt.Errorf("resolver symlinks de projectDir: %w", err)
	}
	real, err = filepath.Abs(real)
	if err != nil {
		return Dir{}, fmt.Errorf("normalizar projectDir real: %w", err)
	}
	return Dir{path: clean, real: filepath.Clean(real)}, nil
}

// FromPath convierte una entrada de CLI en absoluta antes de validarla.
func FromPath(path string) (Dir, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Dir{}, fmt.Errorf("hacer absoluto projectDir: %w", err)
	}
	return Resolve(absolute)
}

func FromCurrent() (Dir, error) {
	current, err := os.Getwd()
	if err != nil {
		return Dir{}, fmt.Errorf("obtener directorio actual: %w", err)
	}
	return Resolve(current)
}

func (d Dir) Path() string {
	return d.path
}

func Compare(left, right Dir) bool {
	if left.path == "" || right.path == "" {
		return false
	}
	if pathsEqual(left.path, right.path) {
		return true
	}
	return left.real != "" && right.real != "" && pathsEqual(left.real, right.real)
}

type OptionalDir struct {
	dir     Dir
	present bool
}

func Some(dir Dir) OptionalDir {
	return OptionalDir{dir: dir, present: true}
}

func (d OptionalDir) Present() bool {
	return d.present
}

func (d OptionalDir) Path() string {
	if !d.present {
		return ""
	}
	return d.dir.Path()
}

func (d OptionalDir) Dir() (Dir, bool) {
	return d.dir, d.present
}
