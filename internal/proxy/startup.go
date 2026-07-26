package proxy

import (
	"context"
	"crypto/rand"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/proc"
)

func Start(ctx context.Context, configured []config.Downstream) (*Service, error) {
	return startService(ctx, configured, func(spec proc.ExecSpec) (proc.ProcessTree, error) {
		return proc.Start(spec)
	})
}

func startService(ctx context.Context, configured []config.Downstream, starter processStarter) (*Service, error) {
	service := &Service{byName: make(map[string]*runtimeDownstream, len(configured))}
	prefixes := make(map[string]struct{}, len(configured))
	for _, downstream := range configured {
		if _, duplicate := prefixes[downstream.Prefix]; duplicate {
			return nil, diagnostics.NewFault(diagnostics.Conflict, "el startup tiene prefijos duplicados", nil)
		}
		prefixes[downstream.Prefix] = struct{}{}
	}

	entries := make([]catalogEntry, 0, len(configured))
	for _, downstream := range configured {
		if err := ctx.Err(); err != nil {
			service.Kill()
			service.Wait()
			return nil, err
		}
		runtime := &runtimeDownstream{status: DownstreamStatus{Name: downstream.Name, State: StateDisabled, Binary: downstream.Binary}}
		service.downstreams = append(service.downstreams, runtime)
		service.byName[downstream.Name] = runtime
		if !downstream.Enabled {
			continue
		}
		runtime.setState(StateStarting, "")
		resolved, err := proc.ResolveExecutable(downstream.Binary)
		if err != nil {
			runtime.setState(StateUnavailable, "ejecutable no disponible")
			continue
		}
		runtime.mu.Lock()
		runtime.status.Binary = resolved.Path()
		runtime.status.Fallback = resolved.UsedFallback()
		runtime.mu.Unlock()
		spec, err := proc.NewExecSpec(resolved, downstream.Args, downstream.Env)
		if err != nil {
			runtime.setState(StateUnavailable, "entorno no disponible")
			continue
		}
		startupCtx, cancel := context.WithTimeout(ctx, ProbeTimeout)
		process, tools, err := startManagedProcess(startupCtx, spec, starter, func(error) {
			runtime.setState(StateUnavailable, "downstream terminó durante runtime")
		})
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				service.Kill()
				service.Wait()
				return nil, ctx.Err()
			}
			runtime.setState(StateUnavailable, "handshake MCP fallido")
			continue
		}
		runtime.mu.Lock()
		runtime.process = process
		if runtime.status.State == StateStarting {
			runtime.status.State = StateAvailable
			runtime.status.Diagnostic = ""
		}
		runtime.mu.Unlock()
		entries = append(entries, catalogEntry{config: downstream, tools: tools})
	}
	if err := ctx.Err(); err != nil {
		service.Kill()
		service.Wait()
		return nil, err
	}

	catalog, err := buildCatalog(entries, rand.Reader)
	if err != nil {
		service.Kill()
		service.Wait()
		return nil, diagnostics.NewFault(diagnostics.Conflict, "no se pudo construir el catálogo global", err)
	}
	service.catalog = catalog
	return service, nil
}
