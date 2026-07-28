package cli

import (
	"context"

	"mcp-gateway/internal/endpoint"
)

type serveApplication interface {
	Serve(context.Context, *endpoint.Port) error
}

func runServe(ctx context.Context, application serveApplication, command parsedCommand) error {
	return application.Serve(ctx, command.port)
}
