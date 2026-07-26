package cli

import (
	"bytes"
	"context"
	"testing"

	"mcp-gateway/internal/app"
)

type workflowApplication struct {
	fakeApplication
	discovery app.DiscoveryResult
	add       app.AddRequest
	removed   string
	enabled   *bool
	changed   bool
}

func (f *workflowApplication) Discover(context.Context, bool) (app.DiscoveryResult, error) {
	return f.discovery, nil
}

func (f *workflowApplication) Add(_ context.Context, request app.AddRequest) error {
	f.add = request
	return nil
}

func (f *workflowApplication) Remove(_ context.Context, name string) (bool, error) {
	f.removed = name
	return f.changed, nil
}

func (f *workflowApplication) SetEnabled(_ context.Context, _ string, enabled bool) (bool, error) {
	f.enabled = &enabled
	return f.changed, nil
}

func TestDiscoverCLIReportsNoInstallationsAsSuccess(t *testing.T) {
	application := &workflowApplication{}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"discover"}, Streams{Out: &stdout, Err: &stderr}, application)
	if code != 0 || stderr.Len() != 0 || !bytes.Contains(stdout.Bytes(), []byte("No se encontraron")) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
