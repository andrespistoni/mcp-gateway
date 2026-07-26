package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/diagnostics"
)

type claudeApplication struct {
	fakeApplication
	projectRequest app.RegisterProjectRequest
	projectResult  app.RegisterProjectResult
	projectErr     error
	installRequest app.InstallClaudeRequest
	installResult  app.InstallClaudeResult
	installErr     error
}

func (f *claudeApplication) RegisterProject(_ context.Context, request app.RegisterProjectRequest) (app.RegisterProjectResult, error) {
	f.projectRequest = request
	return f.projectResult, f.projectErr
}

func (f *claudeApplication) InstallClaude(_ context.Context, request app.InstallClaudeRequest) (app.InstallClaudeResult, error) {
	f.installRequest = request
	return f.installResult, f.installErr
}

func TestRegisterProjectCLIPropagaFlagsYMensajes(t *testing.T) {
	tests := []struct {
		name   string
		result app.RegisterProjectResult
		text   string
	}{
		{name: "creado", result: app.RegisterProjectResult{Created: true, Changed: true}, text: "se creó"},
		{name: "actualizado", result: app.RegisterProjectResult{Updated: true, Changed: true}, text: "se actualizó"},
		{name: "idéntico", result: app.RegisterProjectResult{Updated: true}, text: "no se requirieron cambios"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := &claudeApplication{projectResult: test.result}
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), []string{"register-project", "--project-dir", "/tmp/project", "--port", "4444"}, Streams{Out: &stdout, Err: &stderr}, application)
			if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), test.text) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if application.projectRequest.ProjectDir != "/tmp/project" || application.projectRequest.Port == nil || application.projectRequest.Port.Number() != 4444 {
				t.Fatalf("request = %#v", application.projectRequest)
			}
		})
	}
}

func TestInstallClaudeCLIIdempotenciaYError(t *testing.T) {
	for _, test := range []struct {
		installed bool
		text      string
	}{
		{installed: true, text: "quedó registrado"},
		{installed: false, text: "registro idéntico"},
	} {
		application := &claudeApplication{installResult: app.InstallClaudeResult{Installed: test.installed}}
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), []string{"install-claude", "--port", "5555"}, Streams{Out: &stdout, Err: &stderr}, application)
		if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), test.text) || application.installRequest.Port.Number() != 5555 {
			t.Fatalf("code=%d stdout=%q stderr=%q request=%#v", code, stdout.String(), stderr.String(), application.installRequest)
		}
	}
	application := &claudeApplication{installErr: diagnostics.NewFault(diagnostics.Conflict, "registro incompatible", errors.New("privado"))}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"install-claude"}, Streams{Out: &stdout, Err: &stderr}, application)
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "registro incompatible") || strings.Contains(stderr.String(), "privado") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
