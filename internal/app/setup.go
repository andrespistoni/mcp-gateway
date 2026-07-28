package app

import (
	"context"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
)

// SetupRequest contains only the explicit command-line override. The stored
// port remains authoritative when no override is supplied.
type SetupRequest struct {
	Port *endpoint.Port
}

type SetupResult struct {
	Port      endpoint.Port
	Discovery DiscoveryResult
}

// Setup creates or updates the document, merges only newly discovered
// downstreams, and then converges the managed user service on the final port.
func (r *Runtime) Setup(ctx context.Context, request SetupRequest) (SetupResult, error) {
	repository, ok := r.config.(configUpdater)
	if !ok {
		return SetupResult{}, diagnostics.NewFault(diagnostics.Configuration, "el repositorio no admite mutaciones", nil)
	}

	change, err := repository.Update(ctx, func(document *config.Document) error {
		if request.Port != nil {
			document.Port = *request.Port
		}
		return nil
	})
	if err != nil {
		return SetupResult{}, err
	}
	port := change.Snapshot.Port()

	discovered, err := r.Discover(ctx, true)
	if err != nil {
		return SetupResult{}, err
	}
	if err := r.EnableDaemon(ctx, EnableDaemonRequest{Port: &port}); err != nil {
		return SetupResult{}, err
	}
	return SetupResult{Port: port, Discovery: discovered}, nil
}
