package cli

import (
	"context"
	"fmt"
	"io"
)

type manageApplication interface {
	Remove(context.Context, string) (bool, error)
	SetEnabled(context.Context, string, bool) (bool, error)
}

func writeManage(ctx context.Context, output io.Writer, application manageApplication, command parsedCommand) error {
	var (
		changed bool
		err     error
	)
	switch command.name {
	case "remove":
		changed, err = application.Remove(ctx, command.argument)
	case "enable":
		changed, err = application.SetEnabled(ctx, command.argument, true)
	case "disable":
		changed, err = application.SetEnabled(ctx, command.argument, false)
	}
	if err != nil {
		return err
	}
	state := "sin cambios"
	if changed {
		state = "actualizado"
	}
	_, err = fmt.Fprintf(output, "%s %s: %s.\n", command.name, command.argument, state)
	return err
}
