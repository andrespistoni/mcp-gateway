package proxy

import (
	"context"
	"encoding/json"
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

	p.runCalls()
}

func (p *managedProcess) runCalls() {
	var queued []*call
	var active *call
	correlations := make(map[string]*call)
	nextID := int64(10000)
	finishAll := func(code int64, message string) {
		if active != nil {
			p.finish(active, rpcFailure(active.upstreamID, code, message))
			active = nil
		}
		for _, request := range queued {
			p.finish(request, rpcFailure(request.upstreamID, code, message))
		}
		queued = nil
		for _, request := range correlations {
			p.finish(request, rpcFailure(request.upstreamID, code, message))
		}
	}
	startNext := func() bool {
		for active == nil && len(queued) > 0 {
			request := queued[0]
			queued = queued[1:]
			if request.ctx.Err() != nil {
				p.finish(request, rpcFailure(request.upstreamID, -32003, "Operation cancelled or deadline exceeded"))
				continue
			}
			nextID++
			request.internalID = mcp.NumberID(nextID)
			outbound, err := mcp.NewRequest(request.internalID, "tools/call", json.RawMessage(request.params))
			if err != nil {
				p.finish(request, rpcFailure(request.upstreamID, -32602, "Invalid params"))
				continue
			}
			started, err := p.writeCall(request.ctx, outbound)
			if err != nil {
				p.finish(request, rpcFailure(request.upstreamID, -32003, "Operation cancelled or deadline exceeded"))
				if started {
					_ = p.tree.Kill()
					finishAll(-32002, "Downstream unavailable")
					return false
				}
				continue
			}
			active = request
			correlations[string(request.internalID.Bytes())] = request
		}
		return true
	}

	for {
		if !startNext() {
			return
		}
		select {
		case <-p.stop:
			finishAll(-32003, "Operation cancelled or deadline exceeded")
			return
		case request := <-p.calls:
			if request != nil {
				queued = append(queued, request)
			}
		case request := <-p.cancellations:
			if request == nil {
				continue
			}
			if active == request {
				delete(correlations, string(request.internalID.Bytes()))
				p.finish(request, rpcFailure(request.upstreamID, -32003, "Operation cancelled or deadline exceeded"))
				active = nil
				_ = p.tree.Kill()
				finishAll(-32002, "Downstream unavailable")
				return
			}
			for index, candidate := range queued {
				if candidate == request {
					queued = append(queued[:index], queued[index+1:]...)
					p.finish(request, rpcFailure(request.upstreamID, -32003, "Operation cancelled or deadline exceeded"))
					break
				}
			}
		case result := <-p.inbound:
			if result.err != nil {
				finishAll(-32002, "Downstream unavailable")
				return
			}
			switch result.envelope.Kind() {
			case mcp.Result, mcp.Error:
				id, _ := result.envelope.ID()
				request := correlations[string(id.Bytes())]
				if request == nil { // unknown, duplicate, expired, or late
					continue
				}
				delete(correlations, string(id.Bytes()))
				if active == request {
					active = nil
				}
				p.finish(request, restoreID(result.envelope, request.upstreamID))
			case mcp.Request:
				id, _ := result.envelope.ID()
				reply, _ := mcp.NewError(id, mcp.RPCError{Code: -32601, Message: "Method not found"})
				if p.write(context.Background(), reply) != nil {
					finishAll(-32002, "Downstream unavailable")
					return
				}
			case mcp.Notification:
				// Downstream notifications are intentionally not forwarded in v1.
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

// writeCall distinguishes cancellation while still queued from cancellation
// after the stdin writer may have modified the stream.
func (p *managedProcess) writeCall(ctx context.Context, envelope mcp.Envelope) (bool, error) {
	request := writeRequest{envelope: envelope, started: make(chan struct{}), done: make(chan error, 1)}
	select {
	case p.requests <- request:
	case <-ctx.Done():
		return false, ctx.Err()
	case <-p.stop:
		return false, fmt.Errorf("downstream detenido")
	}
	select {
	case <-request.started:
	case <-ctx.Done():
		return false, ctx.Err()
	case <-p.stop:
		return false, fmt.Errorf("downstream detenido")
	}
	select {
	case err := <-request.done:
		return true, err
	case <-ctx.Done():
		return true, ctx.Err()
	case <-p.stop:
		return true, fmt.Errorf("downstream detenido")
	}
}
