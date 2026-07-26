package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"mcp-gateway/internal/app"
)

type addApplication interface {
	Add(context.Context, app.AddRequest) error
}

func writeAdd(ctx context.Context, streams Streams, application addApplication, command parsedCommand) error {
	environment := make(map[string]string, len(command.environment))
	for _, entry := range command.environment {
		key, value, _ := strings.Cut(entry, "=")
		environment[key] = value
	}
	request := app.AddRequest{
		Name: command.argument, Prefix: command.prefix, Binary: command.binary,
		Args: command.args, Environment: environment, ProjectArgument: command.injectProject,
		Disabled: command.disabled, SkipValidation: command.skipValidation,
	}
	if command.skipValidation {
		if _, err := io.WriteString(streams.Err, "advertencia: se guardará sin verificar y puede quedar no disponible\n"); err != nil {
			return err
		}
	}
	if err := application.Add(ctx, request); err != nil {
		return err
	}
	_, err := fmt.Fprintf(streams.Out, "Downstream %s añadido.\n", command.argument)
	return err
}
