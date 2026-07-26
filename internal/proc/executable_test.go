package proc

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func makeExecutable(t *testing.T, directory, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("fake"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveExecutablePATHExplicitHomeAndFallback(t *testing.T) {
	directory := t.TempDir()
	home := t.TempDir()
	name := "gateway-test-tool"
	pathName := name
	if runtime.GOOS == "windows" {
		pathName += ".exe"
	}
	pathExecutable := makeExecutable(t, directory, name)
	homeExecutable := makeExecutable(t, home, "home-tool")
	t.Setenv("PATH", directory)
	if runtime.GOOS == "windows" {
		t.Setenv("PATHEXT", ".EXE")
		t.Setenv("USERPROFILE", home)
	}
	t.Setenv("HOME", home)

	resolved, err := ResolveExecutable(pathName)
	if err != nil || resolved.Source() != SourcePATH || resolved.Path() != pathExecutable {
		t.Fatalf("PATH = %#v, %v", resolved, err)
	}
	resolved, err = ResolveExecutable(pathExecutable)
	if err != nil || resolved.Source() != SourceExplicit {
		t.Fatalf("explicit = %#v, %v", resolved, err)
	}
	homeConfigured := "~/home-tool"
	if runtime.GOOS == "windows" {
		homeConfigured += ".exe"
	}
	resolved, err = ResolveExecutable(homeConfigured)
	if err != nil || resolved.Path() != homeExecutable {
		t.Fatalf("home = %#v, %v", resolved, err)
	}
	missing := filepath.Join(t.TempDir(), pathName)
	resolved, err = ResolveExecutable(missing)
	if err != nil || !resolved.UsedFallback() || resolved.Path() != pathExecutable {
		t.Fatalf("fallback = %#v, %v", resolved, err)
	}
}

func TestExistingInvalidFileDoesNotFallback(t *testing.T) {
	pathDirectory := t.TempDir()
	configuredDirectory := t.TempDir()
	name := "same-name"
	pathName := name
	if runtime.GOOS == "windows" {
		pathName += ".exe"
	}
	_ = makeExecutable(t, pathDirectory, name)
	t.Setenv("PATH", pathDirectory)
	if runtime.GOOS == "windows" {
		t.Setenv("PATHEXT", ".EXE")
	}
	existingDirectory := filepath.Join(configuredDirectory, pathName)
	if err := os.Mkdir(existingDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveExecutable(existingDirectory); err == nil {
		t.Fatal("un directorio existente no debe activar fallback")
	}
	if runtime.GOOS != "windows" {
		nonExecutable := filepath.Join(configuredDirectory, "non-exec")
		if err := os.WriteFile(nonExecutable, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ResolveExecutable(nonExecutable); err == nil {
			t.Fatal("archivo no ejecutable aceptado")
		}
	}
}

func TestExecSpecKeepsArgvSeparateAndResolvesEnvironment(t *testing.T) {
	directory := t.TempDir()
	executable := makeExecutable(t, directory, "tool")
	resolved, err := ResolveExecutable(executable)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("BASE_VALUE", "valor secreto")
	spec, err := NewExecSpec(resolved, []string{"argumento con espacios", "$(no-shell)"}, map[string]string{
		"COPIED": "prefix-${BASE_VALUE}",
	})
	if err != nil {
		t.Fatal(err)
	}
	args := spec.Args()
	if len(args) != 2 || args[0] != "argumento con espacios" || args[1] != "$(no-shell)" {
		t.Fatalf("argv = %#v", args)
	}
	if !containsEnvironment(spec.Environment(), "COPIED=prefix-valor secreto") {
		t.Fatal("referencia de entorno no expandida")
	}
	args[0] = "mutado"
	if spec.Args()[0] == "mutado" {
		t.Fatal("ExecSpec expone args mutables")
	}
}

func TestExecSpecRejectsInvalidAndDoesNotExposeMissingValue(t *testing.T) {
	directory := t.TempDir()
	executable := makeExecutable(t, directory, "tool")
	resolved, _ := ResolveExecutable(executable)
	if _, err := NewExecSpec(resolved, []string{"bad\x00arg"}, nil); err == nil {
		t.Fatal("NUL en argv aceptado")
	}
	if _, err := NewExecSpec(resolved, nil, map[string]string{"BAD-KEY": "x"}); err == nil {
		t.Fatal("clave env inválida aceptada")
	}
	const missing = "MCP_GATEWAY_TEST_MISSING_SECRET_TOKEN"
	_ = os.Unsetenv(missing)
	_, err := NewExecSpec(resolved, nil, map[string]string{"VALUE": "${" + missing + "}"})
	var target *MissingEnvironmentReference
	if !errors.As(err, &target) {
		t.Fatalf("error faltante = %v", err)
	}
	if strings.Contains(err.Error(), missing) {
		t.Fatalf("error expuso clave sensible: %v", err)
	}
}

func containsEnvironment(environment []string, expected string) bool {
	for _, value := range environment {
		if value == expected {
			return true
		}
	}
	return false
}
