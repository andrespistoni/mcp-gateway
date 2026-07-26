package proc

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	environmentKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	reference      = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
)

type MissingEnvironmentReference struct{}

func (*MissingEnvironmentReference) Error() string {
	return "falta una variable de entorno requerida"
}

type ExecSpec struct {
	executable  ResolvedExecutable
	args        []string
	environment []string
}

func NewExecSpec(executable ResolvedExecutable, args []string, overrides map[string]string) (ExecSpec, error) {
	if executable.Path() == "" {
		return ExecSpec{}, fmt.Errorf("el ejecutable no está resuelto")
	}
	arguments := append([]string(nil), args...)
	for _, argument := range arguments {
		if strings.ContainsRune(argument, 0) {
			return ExecSpec{}, fmt.Errorf("un argumento contiene NUL")
		}
	}
	base := environmentMap(os.Environ())
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		if !environmentKey.MatchString(key) {
			return ExecSpec{}, fmt.Errorf("clave de entorno inválida")
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := overrides[key]
		if strings.ContainsRune(value, 0) {
			return ExecSpec{}, fmt.Errorf("un valor de entorno contiene NUL")
		}
		expanded, err := expandReferences(value, base)
		if err != nil {
			return ExecSpec{}, err
		}
		base[key] = expanded
	}
	environmentKeys := make([]string, 0, len(base))
	for key := range base {
		environmentKeys = append(environmentKeys, key)
	}
	sort.Strings(environmentKeys)
	environment := make([]string, 0, len(environmentKeys))
	for _, key := range environmentKeys {
		environment = append(environment, key+"="+base[key])
	}
	return ExecSpec{executable: executable, args: arguments, environment: environment}, nil
}

func (s ExecSpec) Executable() ResolvedExecutable {
	return s.executable
}

func (s ExecSpec) Args() []string {
	return append([]string(nil), s.args...)
}

func (s ExecSpec) Environment() []string {
	return append([]string(nil), s.environment...)
}

func environmentMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, content, found := strings.Cut(value, "=")
		if found {
			result[key] = content
		}
	}
	return result
}

func expandReferences(value string, environment map[string]string) (string, error) {
	var missing bool
	expanded := reference.ReplaceAllStringFunc(value, func(match string) string {
		parts := reference.FindStringSubmatch(match)
		resolved, exists := environment[parts[1]]
		if !exists {
			missing = true
			return ""
		}
		return resolved
	})
	if missing {
		return "", &MissingEnvironmentReference{}
	}
	return expanded, nil
}
