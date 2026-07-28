package proxy

import (
	"context"
	"encoding/json"
	"sync"

	"mcp-gateway/internal/mcp"
)

type call struct {
	ctx        context.Context
	cancel     context.CancelFunc
	upstreamID mcp.RawID
	tool       string
	params     json.RawMessage
	result     chan mcp.Envelope
	finished   chan struct{}
	once       sync.Once
	internalID mcp.RawID
}

func (p *managedProcess) cancel(request *call) {
	select {
	case p.cancellations <- request:
	case <-request.finished:
	case <-p.done:
	}
}

func (p *managedProcess) finish(request *call, response mcp.Envelope) {
	request.once.Do(func() {
		request.cancel()
		select {
		case request.result <- response:
		default:
		}
		close(request.finished)
		p.credits <- struct{}{}
	})
}

func rpcFailure(id mcp.RawID, code int64, message string) mcp.Envelope {
	response, _ := mcp.NewError(id, mcp.RPCError{Code: code, Message: message})
	return response
}

func restoreID(response mcp.Envelope, id mcp.RawID) mcp.Envelope {
	if response.Kind() == mcp.Error {
		rpcError, _ := response.RPCError()
		restored, _ := mcp.NewError(id, rpcError)
		return restored
	}
	restored, _ := mcp.NewResult(id, response.Result())
	return restored
}
