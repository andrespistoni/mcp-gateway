package app

import (
	"context"
	"errors"

	"mcp-gateway/internal/claude"
	"mcp-gateway/internal/config"
	"mcp-gateway/internal/discovery"
	"mcp-gateway/internal/proc"
	"mcp-gateway/internal/proxy"
)

type ListStatus string

const (
	StatusAvailable   ListStatus = "available"
	StatusUnavailable ListStatus = "unavailable"
	StatusDisabled    ListStatus = "disabled"
)

type ListItem struct {
	Name       string
	Status     ListStatus
	Prefix     string
	Binary     string
	Fallback   bool
	Diagnostic string
}

type configLoader interface {
	Load(context.Context) (config.Snapshot, error)
}

type Runtime struct {
	config            configLoader
	discovery         discoveryRunner
	probe             func(context.Context, proc.ExecSpec) (proxy.ProbeResult, error)
	mutationCommitted func(context.Context) error
	projectRegistrar  projectRegistrar
	claudeRegistrar   claudeRegistrar
	startProxy        proxyFactory
}

func NewRuntime(repository configLoader) *Runtime {
	return &Runtime{
		config:           repository,
		discovery:        discovery.New(proxy.Prober{}),
		probe:            proxy.Probe,
		projectRegistrar: claude.NewDefaultProjectRegistrar(),
		claudeRegistrar:  claude.NewDefaultCLIRegistrar(),
		startProxy: func(ctx context.Context, downstreams []config.Downstream) (startupProxy, error) {
			return proxy.Start(ctx, downstreams)
		},
	}
}

func NewDefaultRuntime() (*Runtime, error) {
	repository, err := config.NewDefaultRepository()
	if err != nil {
		return nil, err
	}
	return NewRuntime(repository), nil
}

func (r *Runtime) List(ctx context.Context) ([]ListItem, error) {
	snapshot, err := r.config.Load(ctx)
	if err != nil {
		return nil, err
	}
	downstreams := snapshot.Downstreams()
	items := make([]ListItem, 0, len(downstreams))
	for _, downstream := range downstreams {
		item := ListItem{
			Name:   downstream.Name,
			Status: StatusDisabled,
			Prefix: downstream.Prefix,
			Binary: downstream.Binary,
		}
		resolved, resolveErr := proc.ResolveExecutable(downstream.Binary)
		if resolveErr != nil {
			if downstream.Enabled {
				item.Status = StatusUnavailable
			}
			item.Diagnostic = "ejecutable no disponible"
			var resolution *proc.ResolutionError
			if errors.As(resolveErr, &resolution) {
				item.Binary = resolution.Attempted()
			}
			items = append(items, item)
			continue
		}
		item.Binary = resolved.Path()
		item.Fallback = resolved.UsedFallback()
		if !downstream.Enabled {
			items = append(items, item)
			continue
		}
		if _, specErr := proc.NewExecSpec(resolved, downstream.Args, downstream.Env); specErr != nil {
			item.Status = StatusUnavailable
			item.Diagnostic = "entorno no disponible"
			items = append(items, item)
			continue
		}
		item.Status = StatusAvailable
		items = append(items, item)
	}
	return items, nil
}
