package config

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"mcp-gateway/internal/endpoint"
)

var (
	namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	envPattern  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	prefixChars = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func fromRaw(raw rawDocument) (Document, error) {
	if raw.Version == nil {
		return Document{}, fmt.Errorf("version es obligatoria")
	}
	document := NewDocument()
	document.Version = *raw.Version
	if raw.Port != nil {
		port, err := endpoint.NewPort(*raw.Port)
		if err != nil {
			return Document{}, fmt.Errorf("port inválido: %w", err)
		}
		document.Port = port
	}
	for _, value := range raw.Downstreams {
		downstream := Downstream{
			Name:            value.Name,
			Prefix:          value.Prefix,
			Binary:          value.Binary,
			Args:            []string{},
			Enabled:         true,
			Env:             map[string]string{},
			ProjectArgument: "",
		}
		if value.Args != nil {
			downstream.Args = append([]string(nil), (*value.Args)...)
		}
		if value.Enabled != nil {
			downstream.Enabled = *value.Enabled
		}
		if value.Env != nil {
			for key, envValue := range *value.Env {
				downstream.Env[key] = envValue
			}
		}
		if value.InjectProject != nil {
			downstream.InjectProject = *value.InjectProject
		}
		if value.ProjectArgument != nil {
			if err := ValidateProjectArgument(*value.ProjectArgument); err != nil {
				return Document{}, fmt.Errorf("project_argument: %w", err)
			}
			downstream.ProjectArgument = *value.ProjectArgument
		} else if downstream.InjectProject {
			downstream.ProjectArgument = "projectPath"
		}
		document.Downstreams = append(document.Downstreams, downstream)
	}
	return document, nil
}

func Validate(document *Document) error {
	if document.Version != SchemaVersion {
		return fmt.Errorf("version debe ser 1")
	}
	if _, err := endpoint.NewPort(document.Port.Number()); err != nil {
		return fmt.Errorf("port inválido: %w", err)
	}
	names := make(map[string]struct{}, len(document.Downstreams))
	prefixes := make(map[string]struct{}, len(document.Downstreams))
	for i := range document.Downstreams {
		downstream := &document.Downstreams[i]
		if err := ValidateName(downstream.Name); err != nil {
			return fmt.Errorf("downstreams[%d].name: %w", i, err)
		}
		if _, exists := names[downstream.Name]; exists {
			return fmt.Errorf("nombre downstream duplicado: %s", downstream.Name)
		}
		names[downstream.Name] = struct{}{}
		if err := ValidatePrefix(downstream.Prefix); err != nil {
			return fmt.Errorf("downstreams[%d].prefix: %w", i, err)
		}
		if _, exists := prefixes[downstream.Prefix]; exists {
			return fmt.Errorf("prefix downstream duplicado: %s", downstream.Prefix)
		}
		prefixes[downstream.Prefix] = struct{}{}
		if downstream.Binary == "" || strings.ContainsRune(downstream.Binary, 0) {
			return fmt.Errorf("downstreams[%d].binary es inválido", i)
		}
		if downstream.Args == nil {
			downstream.Args = []string{}
		}
		for _, argument := range downstream.Args {
			if strings.ContainsRune(argument, 0) {
				return fmt.Errorf("downstreams[%d].args contiene NUL", i)
			}
		}
		if downstream.Env == nil {
			downstream.Env = map[string]string{}
		}
		for key, value := range downstream.Env {
			if err := ValidateEnvironmentEntry(key, value); err != nil {
				return fmt.Errorf("downstreams[%d].env contiene una entrada inválida", i)
			}
		}
		if downstream.InjectProject && downstream.ProjectArgument == "" {
			downstream.ProjectArgument = "projectPath"
		}
		if downstream.ProjectArgument != "" {
			if err := ValidateProjectArgument(downstream.ProjectArgument); err != nil {
				return fmt.Errorf("downstreams[%d].project_argument: %w", i, err)
			}
		}
	}
	return nil
}

func ValidateName(name string) error {
	if len(name) < 1 || len(name) > 64 || !namePattern.MatchString(name) {
		return fmt.Errorf("debe tener 1..64 bytes ASCII y cumplir [A-Za-z0-9][A-Za-z0-9._-]*")
	}
	return nil
}

func ValidatePrefix(prefix string) error {
	if len(prefix) > 128 || !strings.HasSuffix(prefix, "__") {
		return fmt.Errorf("debe terminar exactamente en __ y ocupar como máximo 128 bytes")
	}
	body := strings.TrimSuffix(prefix, "__")
	if body == "" || strings.HasSuffix(body, "_") || !prefixChars.MatchString(body) {
		return fmt.Errorf("el cuerpo debe ser no vacío y usar letras ASCII, dígitos, _ o -")
	}
	return nil
}

func ValidateEnvironmentEntry(key, value string) error {
	if !envPattern.MatchString(key) || strings.ContainsRune(value, 0) {
		return fmt.Errorf("la entrada de entorno es inválida")
	}
	return nil
}

func ValidateProjectArgument(value string) error {
	if len(value) == 0 || len(value) > 256 || !utf8.ValidString(value) {
		return fmt.Errorf("debe ser una clave UTF-8 no vacía de hasta 256 bytes")
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("no puede contener caracteres de control")
		}
	}
	return nil
}
