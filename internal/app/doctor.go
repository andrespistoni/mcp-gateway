package app

import (
	"context"
	"fmt"
	"os/exec"

	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/proc"
)

type DoctorRequest struct {
	Verbose bool
}

type DoctorReport struct {
	Checks []diagnostics.Check
}

func (r DoctorReport) OK() bool {
	for _, check := range r.Checks {
		if !check.OK() {
			return false
		}
	}
	return true
}

// Doctor is intentionally non-mutating. It never starts a daemon or binds a
// listener: the SSE entry is a static endpoint safety check, while downstream
// handshakes remain subject to the injected probe seam.
func (r *Runtime) Doctor(ctx context.Context, request DoctorRequest) (DoctorReport, error) {
	report := DoctorReport{Checks: make([]diagnostics.Check, 0)}
	snapshot, err := r.config.Load(ctx)
	if err != nil {
		report.Checks = append(report.Checks, diagnostics.Failed("configuración", "la configuración no se pudo validar", err))
		return report, nil
	}
	report.Checks = append(report.Checks, diagnostics.Passed("configuración", "YAML v1 válido"))

	port := snapshot.Port()
	for _, downstream := range snapshot.Downstreams() {
		name := "downstream " + downstream.Name
		if !downstream.Enabled {
			report.Checks = append(report.Checks, diagnostics.Passed(name, "deshabilitado"))
			continue
		}
		executable, resolveErr := proc.ResolveExecutable(downstream.Binary)
		if resolveErr != nil {
			report.Checks = append(report.Checks, diagnostics.Failed(name, "ejecutable no disponible", resolveErr))
			continue
		}
		spec, specErr := proc.NewExecSpec(executable, downstream.Args, downstream.Env)
		if specErr != nil {
			report.Checks = append(report.Checks, diagnostics.Failed(name, "entorno no disponible", specErr))
			continue
		}
		if _, probeErr := r.probe(ctx, spec); probeErr != nil {
			report.Checks = append(report.Checks, diagnostics.Failed(name, "handshake MCP fallido", probeErr))
			continue
		}
		detail := "handshake MCP válido"
		if request.Verbose {
			detail = "handshake MCP válido; ejecutable=" + executable.Path()
		}
		report.Checks = append(report.Checks, diagnostics.Passed(name, detail))
	}

	if r.daemonManager == nil {
		report.Checks = append(report.Checks, diagnostics.Failed("daemon", "gestor de daemon no disponible", fmt.Errorf("manager ausente")))
	} else if status, statusErr := r.daemonManager.Status(ctx); statusErr != nil {
		report.Checks = append(report.Checks, diagnostics.Failed("daemon", "no se pudo consultar el daemon", statusErr))
	} else if !status.Installed {
		report.Checks = append(report.Checks, diagnostics.Failed("daemon", "daemon no instalado", fmt.Errorf("ausente")))
	} else if !status.Running {
		report.Checks = append(report.Checks, diagnostics.Failed("daemon", "daemon detenido", fmt.Errorf("detenido")))
	} else {
		report.Checks = append(report.Checks, diagnostics.Passed("daemon", "instalado y en ejecución"))
	}

	// Constructing the endpoint validates the contractual localhost-only shape
	// without probing or binding the configured port.
	if endpoint.LocalhostURL(port, "/sse", nil) == "" {
		report.Checks = append(report.Checks, diagnostics.Failed("SSE", "endpoint SSE inválido", fmt.Errorf("URL vacía")))
	} else {
		report.Checks = append(report.Checks, diagnostics.Passed("SSE", "endpoint localhost validado sin sondeo"))
	}

	if r.doctorClaude != nil {
		err = r.doctorClaude(ctx)
	} else {
		_, err = exec.LookPath("claude")
	}
	if err != nil {
		report.Checks = append(report.Checks, diagnostics.Failed("Claude", "CLI de Claude no disponible", err))
	} else {
		report.Checks = append(report.Checks, diagnostics.Passed("Claude", "CLI disponible"))
	}
	return report, nil
}
