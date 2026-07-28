package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
)

type s8WorkflowApplication struct {
	fakeApplication
	setup  app.SetupResult
	report app.DoctorReport
}

func (s s8WorkflowApplication) Setup(context.Context, app.SetupRequest) (app.SetupResult, error) {
	return s.setup, nil
}

func (s s8WorkflowApplication) Doctor(context.Context, app.DoctorRequest) (app.DoctorReport, error) {
	return s.report, nil
}

func TestSetupAndDoctorCLIWorkflows(t *testing.T) {
	application := s8WorkflowApplication{
		setup:  app.SetupResult{Port: endpoint.MustPort(4444), Discovery: app.DiscoveryResult{Items: []app.DiscoveryItem{{Name: "fake", Path: "/tmp/fake", Added: true}}}},
		report: app.DoctorReport{Checks: []diagnostics.Check{diagnostics.Passed("configuración", "TOKEN=private")}},
	}
	var stdout, stderr bytes.Buffer
	if code := Run(context.Background(), []string{"setup", "--port", "4444"}, Streams{Out: &stdout, Err: &stderr}, application); code != 0 || !strings.Contains(stdout.String(), "register-project") || stderr.Len() != 0 {
		t.Fatalf("setup code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := Run(context.Background(), []string{"doctor", "--verbose"}, Streams{Out: &stdout, Err: &stderr}, application); code != 0 || strings.Contains(stdout.String(), "private") || !strings.Contains(stdout.String(), diagnostics.Redacted) {
		t.Fatalf("doctor code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	application.report = app.DoctorReport{Checks: []diagnostics.Check{diagnostics.Failed("Claude", "no disponible", errors.New("private"))}}
	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"doctor"}, Streams{Out: &stdout, Err: &stderr}, application); code != 1 || !strings.Contains(stderr.String(), "doctor detectó") {
		t.Fatalf("doctor failure code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
