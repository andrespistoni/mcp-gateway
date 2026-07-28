package cli

import (
	"context"
	"fmt"
	"io"

	"mcp-gateway/internal/app"
)

type setupApplication interface {
	Setup(context.Context, app.SetupRequest) (app.SetupResult, error)
}

func writeSetup(ctx context.Context, output io.Writer, application setupApplication, command parsedCommand) error {
	result, err := application.Setup(ctx, app.SetupRequest{Port: command.port})
	if err != nil {
		return err
	}
	for _, item := range result.Discovery.Items {
		state := "conservado"
		if item.Added {
			state = "añadido"
		}
		if _, err := fmt.Fprintf(output, "%s\t%s\t%s\n", item.Name, state, item.Path); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(output, "Setup completado en localhost:%s. Siguiente paso: mcp-gateway register-project.\n", result.Port.Decimal())
	return err
}
