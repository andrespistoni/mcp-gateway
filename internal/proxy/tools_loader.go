package proxy

import (
	"context"
	"fmt"

	"mcp-gateway/internal/mcp"
)

type requestExchange func(context.Context, mcp.Envelope) (mcp.Envelope, error)

func loadTools(ctx context.Context, exchange requestExchange) ([]mcp.Tool, error) {
	tools := make([]mcp.Tool, 0)
	seenCursors := make(map[string]struct{})
	cursor := ""
	for page := 0; page < MaxPages; page++ {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		requestID := mcp.NumberID(int64(page + 2))
		request, _ := mcp.NewRequest(requestID, "tools/list", params)
		response, err := exchange(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("respuesta tools/list inválida: %w", err)
		}
		if response.Kind() == mcp.Error {
			return nil, fmt.Errorf("tools/list devolvió un error JSON-RPC")
		}
		responseID, ok := response.ID()
		if response.Kind() != mcp.Result || !ok || !responseID.Equal(requestID) {
			return nil, fmt.Errorf("respuesta tools/list no correlacionada")
		}
		pageTools, next, err := decodeToolsPage(response.Result())
		if err != nil {
			return nil, err
		}
		if len(tools)+len(pageTools) > MaxTools {
			return nil, fmt.Errorf("tools/list supera 5000 tools")
		}
		tools = append(tools, pageTools...)
		if next == "" {
			return tools, nil
		}
		if _, duplicate := seenCursors[next]; duplicate {
			return nil, fmt.Errorf("tools/list contiene un ciclo de cursor")
		}
		seenCursors[next] = struct{}{}
		cursor = next
	}
	return nil, fmt.Errorf("tools/list supera 100 páginas")
}
