package cli

import (
	"context"
	"fmt"
	"io"

	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/version"
)

type Streams struct {
	Out io.Writer
	Err io.Writer
}

func Run(ctx context.Context, args []string, streams Streams, application Application) int {
	command, err := parse(args)
	if err != nil {
		_, _ = fmt.Fprintln(streams.Err, "error:", diagnostics.RedactText(err.Error()))
		_, _ = fmt.Fprintln(streams.Err, "use 'mcp-gateway help' para ver la sintaxis")
		return 2
	}
	switch command.name {
	case "help":
		if err := writeHelp(streams.Out); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error: no se pudo escribir la ayuda")
			return 1
		}
		return 0
	case "version":
		if _, err := fmt.Fprintln(streams.Out, version.Current().String()); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error: no se pudo escribir la versión")
			return 1
		}
		return 0
	case "list":
		if err := writeList(ctx, streams.Out, application); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error:", diagnostics.ExternalMessage(err))
			return 1
		}
		return 0
	case "discover":
		discoverer, ok := application.(discoverApplication)
		if !ok {
			return unavailableCommand(streams.Err, command.name)
		}
		if err := writeDiscover(ctx, streams.Out, discoverer, command.write); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error:", diagnostics.ExternalMessage(err))
			return 1
		}
		return 0
	case "add":
		adder, ok := application.(addApplication)
		if !ok {
			return unavailableCommand(streams.Err, command.name)
		}
		if err := writeAdd(ctx, streams, adder, command); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error:", diagnostics.ExternalMessage(err))
			return 1
		}
		return 0
	case "remove", "enable", "disable":
		manager, ok := application.(manageApplication)
		if !ok {
			return unavailableCommand(streams.Err, command.name)
		}
		if err := writeManage(ctx, streams.Out, manager, command); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error:", diagnostics.ExternalMessage(err))
			return 1
		}
		return 0
	case "register-project":
		registrar, ok := application.(registerProjectApplication)
		if !ok {
			return unavailableCommand(streams.Err, command.name)
		}
		if err := writeRegisterProject(ctx, streams.Out, registrar, command); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error:", diagnostics.ExternalMessage(err))
			return 1
		}
		return 0
	case "install-claude":
		registrar, ok := application.(installClaudeApplication)
		if !ok {
			return unavailableCommand(streams.Err, command.name)
		}
		if err := writeInstallClaude(ctx, streams.Out, registrar, command); err != nil {
			_, _ = fmt.Fprintln(streams.Err, "error:", diagnostics.ExternalMessage(err))
			return 1
		}
		return 0
	default:
		return unavailableCommand(streams.Err, command.name)
	}
}

func unavailableCommand(output io.Writer, name string) int {
	_, _ = fmt.Fprintf(output, "error: comando %q todavía no implementado\n", name)
	return 1
}
