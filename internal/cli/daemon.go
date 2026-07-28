package cli

import (
	"context"
	"fmt"
	"io"

	"mcp-gateway/internal/app"
)

type daemonApplication interface {
	EnableDaemon(context.Context, app.EnableDaemonRequest) error
	DisableDaemon(context.Context) error
	Restart(context.Context) error
}

func writeDaemon(ctx context.Context, output io.Writer, application daemonApplication, command parsedCommand) error {
	var err error
	switch command.name {
	case "enable-daemon":
		err = application.EnableDaemon(ctx, app.EnableDaemonRequest{Port: command.port})
	case "disable-daemon":
		err = application.DisableDaemon(ctx)
	case "restart":
		err = application.Restart(ctx)
	default:
		return fmt.Errorf("comando de daemon inválido")
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "%s: completado.\n", command.name)
	return err
}
