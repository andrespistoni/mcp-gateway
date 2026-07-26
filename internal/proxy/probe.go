package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/proc"
)

const (
	ProbeTimeout = 5 * time.Second
	MaxPages     = 100
	MaxTools     = 5000
)

type ProbeResult struct {
	Tools []mcp.Tool
}

func Probe(ctx context.Context, spec proc.ExecSpec) (ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()
	tree, err := proc.Start(spec)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("no se pudo iniciar el candidato MCP: %w", err)
	}
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			_ = tree.Terminate()
			done := make(chan struct{})
			go func() {
				_ = tree.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				_ = tree.Kill()
				<-done
			}
		})
	}
	watchDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = tree.Kill()
		case <-watchDone:
		}
	}()
	defer close(watchDone)
	stdin := tree.Stdin()
	stdout := tree.Stdout()
	stderr := tree.Stderr()
	stderrDone := make(chan struct{})
	go func() {
		defer stderr.Close()
		_, _ = io.Copy(io.Discard, stderr)
		close(stderrDone)
	}()
	defer func() {
		cleanup()
		_ = stdin.Close()
		_ = stdout.Close()
		<-stderrDone
	}()

	codec := mcp.NewCodec(stdout, stdin)
	initializeID := mcp.NumberID(1)
	initialize, _ := mcp.NewRequest(initializeID, "initialize", map[string]any{
		"protocolVersion": mcp.ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "mcp-gateway", "version": "probe"},
	})
	if err := codec.Write(initialize); err != nil {
		return ProbeResult{}, probeError(ctx, "no se pudo enviar initialize", err)
	}
	response, err := codec.Read()
	if err != nil {
		return ProbeResult{}, probeError(ctx, "respuesta initialize inválida", err)
	}
	if err := validateInitialize(response, initializeID); err != nil {
		return ProbeResult{}, err
	}
	initialized, _ := mcp.NewNotification("notifications/initialized", map[string]any{})
	if err := codec.Write(initialized); err != nil {
		return ProbeResult{}, probeError(ctx, "no se pudo enviar initialized", err)
	}

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
		if err := codec.Write(request); err != nil {
			return ProbeResult{}, probeError(ctx, "no se pudo enviar tools/list", err)
		}
		response, err := codec.Read()
		if err != nil {
			return ProbeResult{}, probeError(ctx, "respuesta tools/list inválida", err)
		}
		if response.Kind() == mcp.Error {
			return ProbeResult{}, fmt.Errorf("tools/list devolvió un error JSON-RPC")
		}
		responseID, ok := response.ID()
		if response.Kind() != mcp.Result || !ok || !responseID.Equal(requestID) {
			return ProbeResult{}, fmt.Errorf("respuesta tools/list no correlacionada")
		}
		pageTools, next, err := decodeToolsPage(response.Result())
		if err != nil {
			return ProbeResult{}, err
		}
		if len(tools)+len(pageTools) > MaxTools {
			return ProbeResult{}, fmt.Errorf("tools/list supera 5000 tools")
		}
		tools = append(tools, pageTools...)
		if next == "" {
			return ProbeResult{Tools: tools}, nil
		}
		if _, duplicate := seenCursors[next]; duplicate {
			return ProbeResult{}, fmt.Errorf("tools/list contiene un ciclo de cursor")
		}
		seenCursors[next] = struct{}{}
		cursor = next
	}
	return ProbeResult{}, fmt.Errorf("tools/list supera 100 páginas")
}

func decodeToolsPage(raw json.RawMessage) ([]mcp.Tool, string, error) {
	fields, err := mcp.DecodeObject(raw)
	if err != nil {
		return nil, "", fmt.Errorf("página tools/list inválida")
	}
	rawTools, exists := fields["tools"]
	if !exists {
		return nil, "", fmt.Errorf("página tools/list sin tools")
	}
	var definitions []json.RawMessage
	if err := json.Unmarshal(rawTools, &definitions); err != nil || definitions == nil {
		return nil, "", fmt.Errorf("tools debe ser un array")
	}
	tools := make([]mcp.Tool, 0, len(definitions))
	for _, definition := range definitions {
		tool, err := mcp.ParseTool(definition)
		if err != nil {
			return nil, "", err
		}
		tools = append(tools, tool)
	}
	next := ""
	if rawCursor, exists := fields["nextCursor"]; exists && string(rawCursor) != "null" {
		if err := json.Unmarshal(rawCursor, &next); err != nil {
			return nil, "", fmt.Errorf("nextCursor debe ser string")
		}
	}
	return tools, next, nil
}

func probeError(ctx context.Context, message string, cause error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("probe MCP cancelado o expirado: %w", ctx.Err())
	}
	return fmt.Errorf("%s: %w", message, cause)
}

type Prober struct{}

func (Prober) Probe(ctx context.Context, spec proc.ExecSpec) error {
	_, err := Probe(ctx, spec)
	return err
}
