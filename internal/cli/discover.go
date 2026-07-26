package cli

import (
	"context"
	"fmt"
	"io"

	"mcp-gateway/internal/app"
)

type discoverApplication interface {
	Discover(context.Context, bool) (app.DiscoveryResult, error)
}

func writeDiscover(ctx context.Context, output io.Writer, application discoverApplication, write bool) error {
	result, err := application.Discover(ctx, write)
	if err != nil {
		return err
	}
	for _, attempt := range result.Attempts {
		if attempt.Failure == "" {
			continue
		}
		if _, err := fmt.Fprintf(output, "%s\t%s\t%s\n", attempt.Recipe, attempt.Candidate, attempt.Failure); err != nil {
			return err
		}
	}
	if len(result.Items) == 0 {
		_, err = fmt.Fprintln(output, "No se encontraron servidores MCP válidos.")
		return err
	}
	for _, item := range result.Items {
		state := "encontrado"
		if write && item.Added {
			state = "añadido"
		} else if write {
			state = "conservado"
		}
		if _, err := fmt.Fprintf(output, "%s\t%s\t%s\n", item.Name, state, item.Path); err != nil {
			return err
		}
	}
	return nil
}
