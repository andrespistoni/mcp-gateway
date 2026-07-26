package proxy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestServiceStartupStatesPaginationAndSingleInitialize(t *testing.T) {
	binary := buildFake(t)
	events := filepath.Join(t.TempDir(), "events")
	t.Setenv("REQUIRED_RUNTIME_VALUE", "available")
	configured := []config.Downstream{
		downstream("disabled", "disabled__", binary, fakemcp.RuntimeHealthy, false),
		{Name: "missing", Prefix: "missing__", Binary: filepath.Join(t.TempDir(), "absent"), Enabled: true, Env: map[string]string{}},
		downstream("healthy", "healthy__", binary, fakemcp.RuntimeHealthy, true),
		downstream("paged", "paged__", binary, fakemcp.RuntimePaginated, true),
		downstream("crlf", "crlf__", binary, fakemcp.RuntimeCRLF, true),
		downstream("bad-env", "env__", binary, fakemcp.RuntimeHealthy, true),
	}
	configured[2].Args = append(configured[2].Args, "--events="+events)
	configured[5].Env = map[string]string{"TOKEN": "${MISSING_RUNTIME_VALUE}"}
	service, err := Start(context.Background(), configured)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownService(t, service) })
	statuses := service.Statuses()
	want := []State{StateDisabled, StateUnavailable, StateAvailable, StateAvailable, StateAvailable, StateUnavailable}
	for index := range want {
		if statuses[index].State != want[index] {
			t.Fatalf("estado[%d]=%s, want %s", index, statuses[index].State, want[index])
		}
	}
	names := toolNames(service.Catalog())
	if strings.Join(names, ",") != "healthy__echo,paged__first,paged__second,crlf__echo" {
		t.Fatalf("catálogo = %v", names)
	}
	data, err := os.ReadFile(events)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "initialize\n") != 1 {
		t.Fatalf("eventos = %q", data)
	}
}

func TestStartupCancellationRollsBack(t *testing.T) {
	binary := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	service, err := Start(ctx, []config.Downstream{
		downstream("delayed", "delayed__", binary, fakemcp.Delayed, true),
		downstream("never", "never__", binary, fakemcp.RuntimeHealthy, true),
	})
	if !errors.Is(err, context.DeadlineExceeded) || service != nil {
		t.Fatalf("service=%v err=%v", service, err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatalf("rollback cancelado tardó %s", time.Since(started))
	}
}

func TestRuntimeFailureKeepsCatalogSnapshot(t *testing.T) {
	binary := buildFake(t)
	service, err := Start(context.Background(), []config.Downstream{
		downstream("bad", "bad__", binary, fakemcp.RuntimeInvalid, true),
		downstream("good", "good__", binary, fakemcp.RuntimeHealthy, true),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownService(t, service) })
	before := strings.Join(toolNames(service.Catalog()), ",")
	waitForState(t, service, "bad", StateUnavailable)
	after := strings.Join(toolNames(service.Catalog()), ",")
	if before != "bad__echo,good__echo" || after != before {
		t.Fatalf("snapshot cambió: before=%q after=%q", before, after)
	}
	if !errors.Is(service.RequireAvailable("bad"), ErrDownstreamUnavailable) {
		t.Fatal("calls futuras del downstream fallido debían quedar unavailable")
	}
	if err := service.RequireAvailable("good"); err != nil {
		t.Fatalf("downstream saludable afectado: %v", err)
	}
}

func TestRuntimeDrainsAndBoundsStderr(t *testing.T) {
	binary := buildFake(t)
	service, err := Start(context.Background(), []config.Downstream{
		downstream("stderr", "stderr__", binary, fakemcp.RuntimeLargeStderr, true),
	})
	if err != nil {
		t.Fatal(err)
	}
	shutdownService(t, service)
	status := service.Statuses()[0]
	if len(status.Stderr.Data) != StderrLimit || !status.Stderr.Truncated {
		t.Fatalf("stderr bytes=%d truncated=%v", len(status.Stderr.Data), status.Stderr.Truncated)
	}
	if strings.Contains(string(status.Stderr.Data), "healthy__") {
		t.Fatal("stderr se mezcló con protocolo")
	}
}

func TestGlobalCollisionRollsBackStartedProcesses(t *testing.T) {
	binary := buildFake(t)
	started := time.Now()
	service, err := Start(context.Background(), []config.Downstream{
		downstream("short", "a__", binary, fakemcp.CollisionShort, true),
		downstream("long", "a__b__", binary, fakemcp.CollisionLong, true),
	})
	if err == nil || service != nil {
		t.Fatal("la colisión debía abortar el startup")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatalf("rollback tardó %s", time.Since(started))
	}
}

func TestDuplicatePrefixFailsBeforeStartingProcess(t *testing.T) {
	binary := buildFake(t)
	events := filepath.Join(t.TempDir(), "events")
	first := downstream("one", "dup__", binary, fakemcp.RuntimeHealthy, true)
	first.Args = append(first.Args, "--events="+events)
	second := downstream("two", "dup__", binary, fakemcp.RuntimeHealthy, true)
	if service, err := Start(context.Background(), []config.Downstream{first, second}); err == nil || service != nil {
		t.Fatal("el prefijo duplicado debía fallar")
	}
	if _, err := os.Stat(events); !os.IsNotExist(err) {
		t.Fatalf("se inició un proceso antes de validar prefijos: %v", err)
	}
}

func downstream(name, prefix, binary string, scenario fakemcp.Scenario, enabled bool) config.Downstream {
	return config.Downstream{
		Name: name, Prefix: prefix, Binary: binary,
		Args: []string{"--scenario=" + string(scenario)}, Enabled: enabled, Env: map[string]string{},
	}
}

func toolNames(snapshot CatalogSnapshot) []string {
	tools := snapshot.Tools()
	names := make([]string, len(tools))
	for index, tool := range tools {
		names[index] = tool.Name()
	}
	return names
}

func waitForState(t *testing.T, service *Service, name string, state State) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, status := range service.Statuses() {
			if status.Name == name && status.State == state {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s no alcanzó estado %s: %#v", name, state, service.Statuses())
}

func shutdownService(t *testing.T, service *Service) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := service.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}
