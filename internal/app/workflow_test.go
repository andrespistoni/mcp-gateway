package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/daemon"
	"mcp-gateway/internal/discovery"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/proc"
	"mcp-gateway/internal/proxy"
)

func TestSetupMutationAndDoctorWorkflowUsesOnlyFakes(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "fake")
	if runtime.GOOS == "windows" {
		binary += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	if err := os.WriteFile(binary, []byte("fake"), 0o700); err != nil {
		t.Fatal(err)
	}
	manager := &fakeDaemonManager{status: daemon.Status{Installed: true, Running: true}}
	runtime := &Runtime{
		config: testRepository(t),
		discovery: staticDiscovery{result: discovery.Result{Downstreams: []config.Downstream{{
			Name: "fake", Prefix: "fake__", Binary: binary, Args: []string{}, Enabled: true, Env: map[string]string{},
		}}}},
		daemonManager: manager,
		probe: func(context.Context, proc.ExecSpec) (proxy.ProbeResult, error) {
			return proxy.ProbeResult{}, nil
		},
		doctorClaude: func(context.Context) error { return nil },
	}
	port := endpoint.MustPort(4444)
	if _, err := runtime.Setup(context.Background(), SetupRequest{Port: &port}); err != nil {
		t.Fatal(err)
	}
	if changed, err := runtime.SetEnabled(context.Background(), "fake", false); err != nil || !changed {
		t.Fatalf("SetEnabled changed=%v err=%v", changed, err)
	}
	report, err := runtime.Doctor(context.Background(), DoctorRequest{})
	if err != nil || !report.OK() {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if got := manager.calls; len(got) != 2 || got[0] != "enable" || got[1] != "restart" {
		t.Fatalf("daemon calls=%#v", got)
	}
}
