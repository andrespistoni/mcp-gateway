package proxy

import (
	"context"
	"fmt"

	"mcp-gateway/internal/mcp"
)

func (p *managedProcess) runActor(ctx context.Context, startup chan<- startupResult) {
	started := false
	defer func() {
		p.cleanup()
		if started && !p.isIntentional() && p.onFailure != nil {
			p.onFailure(fmt.Errorf("downstream terminó durante runtime"))
		}
		close(p.done)
	}()

	initializeID := mcp.NumberID(1)
	initialize, _ := mcp.NewRequest(initializeID, "initialize", map[string]any{
		"protocolVersion": mcp.ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "mcp-gateway", "version": "runtime"},
	})
	response, err := p.exchange(ctx, initialize)
	if err == nil {
		err = validateInitialize(response, initializeID)
	}
	if err == nil {
		initialized, _ := mcp.NewNotification("notifications/initialized", map[string]any{})
		err = p.write(ctx, initialized)
	}
	var tools []mcp.Tool
	if err == nil {
		tools, err = loadTools(ctx, p.exchange)
	}
	if err != nil {
		startup <- startupResult{err: fmt.Errorf("startup downstream falló: %w", err)}
		return
	}
	started = true
	startup <- startupResult{tools: tools}

	for {
		select {
		case <-p.stop:
			return
		case result := <-p.inbound:
			if result.err != nil {
				return
			}
			switch result.envelope.Kind() {
			case mcp.Notification, mcp.Result, mcp.Error:
				// Runtime notifications and uncorrelated late responses are not forwarded.
			case mcp.Request:
				id, _ := result.envelope.ID()
				reply, _ := mcp.NewError(id, mcp.RPCError{Code: -32601, Message: "Method not found"})
				if p.write(context.Background(), reply) != nil {
					return
				}
			}
		}
	}
}

func (p *managedProcess) exchange(ctx context.Context, envelope mcp.Envelope) (mcp.Envelope, error) {
	if err := p.write(ctx, envelope); err != nil {
		return mcp.Envelope{}, err
	}
	select {
	case result := <-p.inbound:
		return result.envelope, result.err
	case <-ctx.Done():
		return mcp.Envelope{}, ctx.Err()
	case <-p.stop:
		return mcp.Envelope{}, fmt.Errorf("downstream detenido")
	}
}

func (p *managedProcess) write(ctx context.Context, envelope mcp.Envelope) error {
	request := writeRequest{envelope: envelope, done: make(chan error, 1)}
	select {
	case p.requests <- request:
	case <-ctx.Done():
		return ctx.Err()
	case <-p.stop:
		return fmt.Errorf("downstream detenido")
	}
	select {
	case err := <-request.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-p.stop:
		return fmt.Errorf("downstream detenido")
	}
}
