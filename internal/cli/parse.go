package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/endpoint"
)

var publicCommands = []string{
	"setup", "discover", "add", "remove", "enable", "disable", "list", "doctor", "serve",
	"enable-daemon", "disable-daemon", "restart", "register-project", "install-claude", "version", "help",
}

type parsedCommand struct {
	name           string
	argument       string
	port           *endpoint.Port
	write          bool
	verbose        bool
	prefix         string
	binary         string
	args           []string
	environment    []string
	injectProject  string
	disabled       bool
	skipValidation bool
	projectDir     string
}

type repeatedValues []string

func (v *repeatedValues) String() string { return strings.Join(*v, ",") }
func (v *repeatedValues) Set(value string) error {
	*v = append(*v, value)
	return nil
}

type optionalValue struct {
	value string
	set   bool
}

func (v *optionalValue) String() string { return v.value }
func (v *optionalValue) Set(value string) error {
	v.value = value
	v.set = true
	return nil
}

func parse(args []string) (parsedCommand, error) {
	if len(args) == 0 {
		return parsedCommand{}, fmt.Errorf("falta un comando")
	}
	command := parsedCommand{name: args[0]}
	if !isPublicCommand(command.name) {
		return parsedCommand{}, fmt.Errorf("comando desconocido: %s", command.name)
	}
	parseArgs := args[1:]
	if command.name == "add" || command.name == "remove" || command.name == "enable" || command.name == "disable" {
		if len(parseArgs) == 0 {
			return parsedCommand{}, fmt.Errorf("%s requiere exactamente un nombre", command.name)
		}
		if err := config.ValidateName(parseArgs[0]); err != nil {
			return parsedCommand{}, fmt.Errorf("nombre inválido")
		}
		command.argument = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	set := flag.NewFlagSet(command.name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var portValue optionalValue
	var injectProject optionalValue
	var projectDir optionalValue
	switch command.name {
	case "setup", "serve", "enable-daemon", "install-claude":
		set.Var(&portValue, "port", "puerto")
	case "discover":
		set.BoolVar(&command.write, "write", false, "escribir resultados")
	case "add":
		set.StringVar(&command.prefix, "prefix", "", "prefijo")
		set.StringVar(&command.binary, "binary", "", "binario")
		set.Var((*repeatedValues)(&command.args), "arg", "argumento")
		set.Var((*repeatedValues)(&command.environment), "env", "entorno")
		set.Var(&injectProject, "inject-project", "argumento de proyecto")
		set.BoolVar(&command.disabled, "disabled", false, "guardar deshabilitado")
		set.BoolVar(&command.skipValidation, "skip-validation", false, "omitir validación")
	case "doctor":
		set.BoolVar(&command.verbose, "verbose", false, "detalle")
	case "register-project":
		set.Var(&projectDir, "project-dir", "directorio de proyecto")
		set.Var(&portValue, "port", "puerto")
	}
	if err := set.Parse(parseArgs); err != nil {
		return parsedCommand{}, fmt.Errorf("sintaxis inválida para %s", command.name)
	}
	positionals := set.Args()
	if len(positionals) != 0 {
		return parsedCommand{}, fmt.Errorf("%s no admite argumentos posicionales", command.name)
	}
	if command.name == "add" {
		if command.prefix == "" || command.binary == "" {
			return parsedCommand{}, fmt.Errorf("add requiere --prefix y --binary")
		}
		if err := config.ValidatePrefix(command.prefix); err != nil {
			return parsedCommand{}, fmt.Errorf("prefijo inválido")
		}
		if strings.ContainsRune(command.binary, 0) {
			return parsedCommand{}, fmt.Errorf("binario inválido")
		}
		for _, argument := range command.args {
			if strings.ContainsRune(argument, 0) {
				return parsedCommand{}, fmt.Errorf("--arg contiene NUL")
			}
		}
		for _, entry := range command.environment {
			key, value, found := strings.Cut(entry, "=")
			if !found || config.ValidateEnvironmentEntry(key, value) != nil {
				return parsedCommand{}, fmt.Errorf("--env requiere KEY=VALUE")
			}
		}
		if injectProject.set {
			if err := config.ValidateProjectArgument(injectProject.value); err != nil {
				return parsedCommand{}, fmt.Errorf("--inject-project inválido")
			}
			command.injectProject = injectProject.value
		}
	}
	if projectDir.set {
		if projectDir.value == "" || strings.ContainsRune(projectDir.value, 0) {
			return parsedCommand{}, fmt.Errorf("--project-dir inválido")
		}
		command.projectDir = projectDir.value
	}
	if portValue.set {
		port, err := endpoint.ParsePort(portValue.value)
		if err != nil {
			return parsedCommand{}, fmt.Errorf("--port inválido")
		}
		command.port = &port
	}
	return command, nil
}

func isPublicCommand(value string) bool {
	for _, command := range publicCommands {
		if value == command {
			return true
		}
	}
	return false
}
