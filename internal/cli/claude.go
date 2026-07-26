package cli

import (
	"context"
	"fmt"
	"io"

	"mcp-gateway/internal/app"
)

type registerProjectApplication interface {
	RegisterProject(context.Context, app.RegisterProjectRequest) (app.RegisterProjectResult, error)
}

type installClaudeApplication interface {
	InstallClaude(context.Context, app.InstallClaudeRequest) (app.InstallClaudeResult, error)
}

func writeRegisterProject(ctx context.Context, output io.Writer, application registerProjectApplication, command parsedCommand) error {
	result, err := application.RegisterProject(ctx, app.RegisterProjectRequest{ProjectDir: command.projectDir, Port: command.port})
	if err != nil {
		return err
	}
	switch {
	case result.Created:
		_, err = fmt.Fprintln(output, "Proyecto registrado: se creó mcpServers.mcp-gateway.")
	case result.Changed:
		_, err = fmt.Fprintln(output, "Proyecto registrado: se actualizó mcpServers.mcp-gateway.")
	default:
		_, err = fmt.Fprintln(output, "Proyecto ya registrado; no se requirieron cambios.")
	}
	return err
}

func writeInstallClaude(ctx context.Context, output io.Writer, application installClaudeApplication, command parsedCommand) error {
	result, err := application.InstallClaude(ctx, app.InstallClaudeRequest{Port: command.port})
	if err != nil {
		return err
	}
	if result.Installed {
		_, err = fmt.Fprintln(output, "Claude quedó registrado para el usuario.")
	} else {
		_, err = fmt.Fprintln(output, "Claude ya tenía un registro idéntico.")
	}
	return err
}
