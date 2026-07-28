package app

import (
	"context"
	"errors"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/daemon"
)

func TestMutationsRestartOnlyManagedRunningDaemonAfterCommit(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     daemon.Status
		restartErr error
		wantCalls  string
		wantErr    bool
	}{
		{name: "running", status: daemon.Status{Installed: true, Running: true}, wantCalls: "restart"},
		{name: "absent", status: daemon.Status{}, wantCalls: ""},
		{name: "stopped", status: daemon.Status{Installed: true}, wantCalls: ""},
		{name: "restart fails after commit", status: daemon.Status{Installed: true, Running: true}, restartErr: errors.New("private"), wantCalls: "restart", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := testRepository(t)
			_, err := repository.Update(context.Background(), func(document *config.Document) error {
				document.Downstreams = []config.Downstream{{Name: "one", Prefix: "one__", Binary: "one", Args: []string{}, Enabled: true, Env: map[string]string{}}}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			manager := &fakeDaemonManager{status: test.status, restartErr: test.restartErr}
			runtime := &Runtime{config: repository, daemonManager: manager}
			changed, err := runtime.SetEnabled(context.Background(), "one", false)
			if changed != true || (err != nil) != test.wantErr {
				t.Fatalf("changed=%v err=%v", changed, err)
			}
			calls := ""
			for _, call := range manager.calls {
				calls += call
			}
			if calls != test.wantCalls {
				t.Fatalf("calls=%q want %q", calls, test.wantCalls)
			}
			snapshot, loadErr := repository.Load(context.Background())
			if loadErr != nil || snapshot.Downstreams()[0].Enabled {
				t.Fatalf("la mutación confirmada se revirtió: snapshot=%#v err=%v", snapshot, loadErr)
			}
		})
	}
}
