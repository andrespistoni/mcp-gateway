package app

import (
	"context"
	"fmt"

	"mcp-gateway/internal/endpoint"
)

// startGateway acquires resources in serve order. S-5 supplies the concrete
// listener and composition while this layer remains the rollback authority.
func (r *Runtime) startGateway(ctx context.Context, requestedPort *endpoint.Port, acquire listenerFactory, compose startupComposer) (*startedRuntime, endpoint.Port, error) {
	if acquire == nil || compose == nil || r.startProxy == nil {
		return nil, endpoint.Port{}, fmt.Errorf("composición de startup incompleta")
	}
	snapshot, err := r.config.Load(ctx)
	if err != nil {
		return nil, endpoint.Port{}, err
	}
	configuredPort := snapshot.Port()
	port := endpoint.ResolvePort(requestedPort, &configuredPort)
	listener, err := acquire(ctx, port)
	if err != nil {
		return nil, endpoint.Port{}, err
	}
	resources := &startedRuntime{listener: listener}
	service, err := r.startProxy(ctx, snapshot.Downstreams())
	if err != nil {
		_ = listener.Close()
		return nil, endpoint.Port{}, err
	}
	resources.proxy = service
	if err := compose(listener, service); err != nil {
		_ = resources.Close(ctx)
		return nil, endpoint.Port{}, err
	}
	return resources, port, nil
}
