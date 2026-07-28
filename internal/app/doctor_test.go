package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/daemon"
	"mcp-gateway/internal/proc"
	"mcp-gateway/internal/proxy"
)

func TestDoctorUsesInjectedChecksAndNeverBindsListener(t *testing.T) {
	repository := testRepository(t)
	binary := filepath.Join(t.TempDir(), "fake")
	if runtime.GOOS == "windows" {
		binary += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	if err := os.WriteFile(binary, []byte("fake"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := repository.Update(context.Background(), func(document *config.Document) error {
		document.Downstreams = []config.Downstream{{Name: "fake", Prefix: "fake__", Binary: binary, Args: []string{}, Enabled: true, Env: map[string]string{}}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	probed := false
	runtime := &Runtime{
		config:        repository,
		daemonManager: &fakeDaemonManager{status: daemon.Status{Installed: true, Running: true}},
		probe: func(context.Context, proc.ExecSpec) (proxy.ProbeResult, error) {
			probed = true
			return proxy.ProbeResult{}, nil
		},
		doctorClaude: func(context.Context) error { return nil },
	}
	report, err := runtime.Doctor(context.Background(), DoctorRequest{Verbose: true})
	if err != nil || !probed || !report.OK() || len(report.Checks) != 5 {
		t.Fatalf("report=%#v err=%v probed=%v", report, err, probed)
	}
}

func TestDoctorReportsRequiredFailureWithoutLeakingCause(t *testing.T) {
	runtime := &Runtime{config: missingDaemonConfig{}, doctorClaude: func(context.Context) error { return errors.New("TOKEN=private") }}
	report, err := runtime.Doctor(context.Background(), DoctorRequest{})
	if err != nil || report.OK() || len(report.Checks) != 1 || report.Checks[0].Name != "configuración" {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}
