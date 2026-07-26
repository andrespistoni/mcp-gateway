package proc

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ExecutableSource string

const (
	SourceExplicit         ExecutableSource = "explicit"
	SourcePATH             ExecutableSource = "PATH"
	SourceBasenameFallback ExecutableSource = "basename-fallback"
)

type ResolvedExecutable struct {
	path   string
	source ExecutableSource
}

func (r ResolvedExecutable) Path() string {
	return r.path
}

func (r ResolvedExecutable) Source() ExecutableSource {
	return r.source
}

func (r ResolvedExecutable) UsedFallback() bool {
	return r.source == SourceBasenameFallback
}

type ResolutionError struct {
	attempted string
	cause     error
}

func (e *ResolutionError) Error() string {
	return "no se pudo resolver un ejecutable regular y permitido"
}

func (e *ResolutionError) Unwrap() error {
	return e.cause
}

func (e *ResolutionError) Attempted() string {
	return e.attempted
}

func ResolveExecutable(binary string) (ResolvedExecutable, error) {
	if binary == "" || strings.ContainsRune(binary, 0) {
		return ResolvedExecutable{}, &ResolutionError{attempted: binary, cause: fmt.Errorf("binary inválido")}
	}
	expanded, err := expandHome(binary)
	if err != nil {
		return ResolvedExecutable{}, &ResolutionError{attempted: binary, cause: err}
	}
	if !containsPathSeparator(expanded) {
		return resolveFromPATH(expanded, SourcePATH)
	}
	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return ResolvedExecutable{}, &ResolutionError{attempted: expanded, cause: err}
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err == nil {
		if err := validateExecutable(absolute, info); err != nil {
			return ResolvedExecutable{}, &ResolutionError{attempted: absolute, cause: err}
		}
		return ResolvedExecutable{path: absolute, source: SourceExplicit}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ResolvedExecutable{}, &ResolutionError{attempted: absolute, cause: err}
	}
	resolved, fallbackErr := resolveFromPATH(filepath.Base(absolute), SourceBasenameFallback)
	if fallbackErr != nil {
		return ResolvedExecutable{}, &ResolutionError{attempted: absolute, cause: fallbackErr}
	}
	return resolved, nil
}

func resolveFromPATH(name string, source ExecutableSource) (ResolvedExecutable, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return ResolvedExecutable{}, &ResolutionError{attempted: name, cause: err}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ResolvedExecutable{}, &ResolutionError{attempted: path, cause: err}
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return ResolvedExecutable{}, &ResolutionError{attempted: absolute, cause: err}
	}
	if err := validateExecutable(absolute, info); err != nil {
		return ResolvedExecutable{}, &ResolutionError{attempted: absolute, cause: err}
	}
	return ResolvedExecutable{path: filepath.Clean(absolute), source: source}, nil
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

func containsPathSeparator(path string) bool {
	return strings.ContainsRune(path, '/') || strings.ContainsRune(path, '\\')
}
