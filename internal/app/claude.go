package app

import (
	"context"
	"errors"
	"os"

	"mcp-gateway/internal/claude"
	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/project"
)

type projectRegistrar interface {
	Register(context.Context, project.Dir, endpoint.Port) (claude.ProjectRegistration, error)
}

type claudeRegistrar interface {
	Install(context.Context, endpoint.Port) (claude.InstallResult, error)
}

type RegisterProjectRequest struct {
	ProjectDir string
	Port       *endpoint.Port
}

type RegisterProjectResult struct {
	ProjectDir string
	Created    bool
	Updated    bool
	Changed    bool
}

type InstallClaudeRequest struct {
	Port *endpoint.Port
}

type InstallClaudeResult struct {
	Installed bool
}

func (r *Runtime) RegisterProject(ctx context.Context, request RegisterProjectRequest) (RegisterProjectResult, error) {
	var directory project.Dir
	var err error
	if request.ProjectDir == "" {
		directory, err = project.FromCurrent()
	} else {
		directory, err = project.FromPath(request.ProjectDir)
	}
	if err != nil {
		return RegisterProjectResult{}, diagnostics.NewFault(diagnostics.Validation, "projectDir no es válido", err)
	}
	port, err := r.resolvePort(ctx, request.Port)
	if err != nil {
		return RegisterProjectResult{}, err
	}
	registration, err := r.projectRegistrar.Register(ctx, directory, port)
	if err != nil {
		return RegisterProjectResult{}, err
	}
	return RegisterProjectResult{
		ProjectDir: directory.Path(), Created: registration.Created, Updated: registration.Updated, Changed: registration.Changed,
	}, nil
}

func (r *Runtime) InstallClaude(ctx context.Context, request InstallClaudeRequest) (InstallClaudeResult, error) {
	port, err := r.resolvePort(ctx, request.Port)
	if err != nil {
		return InstallClaudeResult{}, err
	}
	result, err := r.claudeRegistrar.Install(ctx, port)
	if err != nil {
		return InstallClaudeResult{}, err
	}
	return InstallClaudeResult{Installed: result.Installed}, nil
}

func (r *Runtime) resolvePort(ctx context.Context, flag *endpoint.Port) (endpoint.Port, error) {
	if flag != nil {
		return endpoint.ResolvePort(flag, nil), nil
	}
	snapshot, err := r.config.Load(ctx)
	if err == nil {
		configured := snapshot.Port()
		return endpoint.ResolvePort(nil, &configured), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return endpoint.ResolvePort(nil, nil), nil
	}
	return endpoint.Port{}, err
}
