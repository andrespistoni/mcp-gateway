package app

import (
	"context"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/discovery"
	"mcp-gateway/internal/proc"
)

type discoveryRunner interface {
	Discover(context.Context) (discovery.Result, error)
}

type configUpdater interface {
	Update(context.Context, func(*config.Document) error) (config.Change, error)
}

type DiscoveryItem struct {
	Name  string
	Path  string
	Added bool
}

type DiscoveryResult struct {
	Items    []DiscoveryItem
	Attempts []discovery.Attempt
}

func (r *Runtime) Discover(ctx context.Context, write bool) (DiscoveryResult, error) {
	found, err := r.discovery.Discover(ctx)
	if err != nil {
		return DiscoveryResult{}, diagnostics.NewFault(diagnostics.Process, "no se pudo completar discovery", err)
	}
	result := DiscoveryResult{Items: make([]DiscoveryItem, len(found.Downstreams)), Attempts: found.Attempts}
	for i, downstream := range found.Downstreams {
		result.Items[i] = DiscoveryItem{Name: downstream.Name, Path: downstream.Binary}
	}
	if !write || len(found.Downstreams) == 0 {
		return result, nil
	}
	repository, ok := r.config.(configUpdater)
	if !ok {
		return DiscoveryResult{}, diagnostics.NewFault(diagnostics.Configuration, "el repositorio no admite mutaciones", nil)
	}
	added := make(map[string]bool)
	_, err = repository.Update(ctx, func(document *config.Document) error {
		for _, candidate := range found.Downstreams {
			nameExists := false
			prefixExists := false
			for _, current := range document.Downstreams {
				nameExists = nameExists || current.Name == candidate.Name
				prefixExists = prefixExists || current.Prefix == candidate.Prefix
			}
			if nameExists || prefixExists {
				continue
			}
			document.Downstreams = append(document.Downstreams, candidate)
			added[candidate.Name] = true
		}
		return nil
	})
	if err != nil {
		return DiscoveryResult{}, err
	}
	for i := range result.Items {
		result.Items[i].Added = added[result.Items[i].Name]
	}
	return result, nil
}

type AddRequest struct {
	Name            string
	Prefix          string
	Binary          string
	Args            []string
	Environment     map[string]string
	ProjectArgument string
	Disabled        bool
	SkipValidation  bool
}

func (r *Runtime) Add(ctx context.Context, request AddRequest) error {
	downstream := config.Downstream{
		Name: request.Name, Prefix: request.Prefix, Binary: request.Binary,
		Args: append([]string(nil), request.Args...), Enabled: !request.Disabled,
		Env: cloneEnvironment(request.Environment), InjectProject: request.ProjectArgument != "",
		ProjectArgument: request.ProjectArgument,
	}
	document := config.NewDocument()
	document.Downstreams = append(document.Downstreams, downstream)
	if err := config.Validate(&document); err != nil {
		return diagnostics.NewFault(diagnostics.Validation, "la definición downstream no es válida", err)
	}
	if !request.SkipValidation {
		executable, err := proc.ResolveExecutable(request.Binary)
		if err != nil {
			return diagnostics.NewFault(diagnostics.Unavailable, "el ejecutable downstream no está disponible", err)
		}
		spec, err := proc.NewExecSpec(executable, request.Args, request.Environment)
		if err != nil {
			return diagnostics.NewFault(diagnostics.Unavailable, "el entorno downstream no está disponible", err)
		}
		if _, err := r.probe(ctx, spec); err != nil {
			return diagnostics.NewFault(diagnostics.Protocol, "la validación MCP falló", err)
		}
	}
	repository, ok := r.config.(configUpdater)
	if !ok {
		return diagnostics.NewFault(diagnostics.Configuration, "el repositorio no admite mutaciones", nil)
	}
	_, err := repository.Update(ctx, func(document *config.Document) error {
		for _, current := range document.Downstreams {
			if current.Name == downstream.Name {
				return diagnostics.NewFault(diagnostics.Conflict, "ya existe un downstream con ese nombre", nil)
			}
			if current.Prefix == downstream.Prefix {
				return diagnostics.NewFault(diagnostics.Conflict, "ya existe un downstream con ese prefijo", nil)
			}
		}
		document.Downstreams = append(document.Downstreams, downstream)
		return nil
	})
	return err
}

func cloneEnvironment(environment map[string]string) map[string]string {
	clone := make(map[string]string, len(environment))
	for key, value := range environment {
		clone[key] = value
	}
	return clone
}
