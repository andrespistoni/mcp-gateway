package cli

import (
	"bytes"
	"context"
	"testing"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/endpoint"
)

type fakeDaemonApplication struct {
	fakeApplication
	enabled   *endpoint.Port
	disabled  bool
	restarted bool
}

func (f *fakeDaemonApplication) EnableDaemon(_ context.Context, request app.EnableDaemonRequest) error {
	f.enabled = request.Port
	return nil
}
func (f *fakeDaemonApplication) DisableDaemon(context.Context) error { f.disabled = true; return nil }
func (f *fakeDaemonApplication) Restart(context.Context) error       { f.restarted = true; return nil }

func TestDaemonCommandsDispatchAndKeepOutputChannel(t *testing.T) {
	application := &fakeDaemonApplication{}
	for _, args := range [][]string{{"enable-daemon", "--port", "4444"}, {"disable-daemon"}, {"restart"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(context.Background(), args, Streams{Out: &stdout, Err: &stderr}, application); code != 0 || stderr.Len() != 0 || !bytes.Contains(stdout.Bytes(), []byte("completado")) {
			t.Fatalf("%v: code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
	if application.enabled == nil || application.enabled.Number() != 4444 || !application.disabled || !application.restarted {
		t.Fatalf("application = %#v", application)
	}
}
