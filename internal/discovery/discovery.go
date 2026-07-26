package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/proc"
)

type Prober interface {
	Probe(context.Context, proc.ExecSpec) error
}

type Attempt struct {
	Recipe    string
	Candidate string
	Path      string
	Failure   string
}

type Result struct {
	Downstreams []config.Downstream
	Attempts    []Attempt
}

type Service struct {
	prober  Prober
	recipes []Recipe
}

func New(prober Prober) *Service {
	return &Service{prober: prober, recipes: Recipes()}
}

func (s *Service) Discover(ctx context.Context) (Result, error) {
	if s == nil || s.prober == nil {
		return Result{}, fmt.Errorf("prober de discovery ausente")
	}
	result := Result{Downstreams: []config.Downstream{}, Attempts: []Attempt{}}
	for _, recipe := range s.recipes {
		seen := make(map[string]struct{})
		for _, candidate := range recipe.BinaryCandidates {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			resolved, err := resolveCandidate(candidate)
			attempt := Attempt{Recipe: recipe.Name, Candidate: candidate}
			if err != nil {
				attempt.Failure = "candidato no disponible"
				result.Attempts = append(result.Attempts, attempt)
				continue
			}
			attempt.Path = resolved.Path()
			canonical := resolved.Path()
			if evaluated, err := filepath.EvalSymlinks(canonical); err == nil {
				canonical = filepath.Clean(evaluated)
			}
			if _, duplicate := seen[canonical]; duplicate {
				attempt.Failure = "candidato duplicado"
				result.Attempts = append(result.Attempts, attempt)
				continue
			}
			seen[canonical] = struct{}{}
			spec, err := proc.NewExecSpec(resolved, recipe.Args, nil)
			if err == nil {
				err = s.prober.Probe(ctx, spec)
			}
			if err != nil {
				attempt.Failure = "handshake MCP fallido"
				result.Attempts = append(result.Attempts, attempt)
				continue
			}
			result.Attempts = append(result.Attempts, attempt)
			result.Downstreams = append(result.Downstreams, config.Downstream{
				Name: recipe.Name, Prefix: recipe.Prefix, Binary: resolved.Path(),
				Args: append([]string(nil), recipe.Args...), Enabled: true, Env: map[string]string{},
				InjectProject: recipe.InjectProject, ProjectArgument: recipe.ProjectArgument,
			})
			break
		}
	}
	return result, nil
}

func resolveCandidate(candidate string) (proc.ResolvedExecutable, error) {
	if !strings.ContainsAny(candidate, `/\`) && candidate != "~" {
		return proc.ResolveExecutable(candidate)
	}
	expanded := candidate
	if candidate == "~" || strings.HasPrefix(candidate, "~/") || strings.HasPrefix(candidate, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return proc.ResolvedExecutable{}, err
		}
		if candidate == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, candidate[2:])
		}
	}
	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return proc.ResolvedExecutable{}, err
	}
	if _, err := os.Stat(absolute); err != nil {
		return proc.ResolvedExecutable{}, err
	}
	return proc.ResolveExecutable(absolute)
}
