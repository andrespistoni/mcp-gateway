package proxy

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"mcp-gateway/internal/proc"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestProbeHandshakePaginationAndFailures(t *testing.T) {
	binary := buildFake(t)
	tests := []struct {
		scenario fakemcp.Scenario
		tools    []string
		fails    bool
	}{
		{scenario: fakemcp.Healthy, tools: []string{"echo"}},
		{scenario: fakemcp.Paginated, tools: []string{"first", "second"}},
		{scenario: fakemcp.CursorCycle, fails: true},
		{scenario: fakemcp.InvalidTools, fails: true},
		{scenario: fakemcp.InvalidCursor, fails: true},
		{scenario: fakemcp.HundredPages, tools: makeNames(100)},
		{scenario: fakemcp.TooManyPages, fails: true},
		{scenario: fakemcp.MaxTools, tools: makeNames(5000)},
		{scenario: fakemcp.TooManyTools, fails: true},
		{scenario: fakemcp.Batch, fails: true},
		{scenario: fakemcp.EmptyLine, fails: true},
		{scenario: fakemcp.PartialEOF, fails: true},
	}
	for _, test := range tests {
		t.Run(string(test.scenario), func(t *testing.T) {
			spec := fakeSpec(t, binary, test.scenario)
			result, err := Probe(context.Background(), spec)
			if test.fails {
				if err == nil {
					t.Fatal("Probe debía fallar")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Tools) != len(test.tools) {
				t.Fatalf("tools = %d", len(result.Tools))
			}
			for i, name := range test.tools {
				if result.Tools[i].Name() != name {
					t.Fatalf("tool[%d] = %q", i, result.Tools[i].Name())
				}
			}
		})
	}
}

func makeNames(count int) []string {
	result := make([]string, count)
	for i := range result {
		result[i] = fmt.Sprintf("tool-%d", i)
	}
	return result
}

func TestProbeCancellationTerminatesCandidate(t *testing.T) {
	binary := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := Probe(ctx, fakeSpec(t, binary, fakemcp.Delayed))
	if err == nil || time.Since(started) > 3*time.Second {
		t.Fatalf("cancelación = %v, duración=%s", err, time.Since(started))
	}
}

func TestProbeDrainsLargeResponseAfterImmediateExit(t *testing.T) {
	binary := buildFake(t)
	for iteration := range 3 {
		result, err := Probe(context.Background(), fakeSpec(t, binary, fakemcp.MaxTools))
		if err != nil {
			t.Fatalf("iteración %d: %v", iteration, err)
		}
		if len(result.Tools) != MaxTools {
			t.Fatalf("iteración %d: tools=%d", iteration, len(result.Tools))
		}
	}
}

func buildFake(t *testing.T) string {
	t.Helper()
	name := "fake-mcp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := fakemcp.Build(ctx, binary); err != nil {
		t.Fatal(err)
	}
	return binary
}

func fakeSpec(t *testing.T, binary string, scenario fakemcp.Scenario) proc.ExecSpec {
	t.Helper()
	executable, err := proc.ResolveExecutable(binary)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := proc.NewExecSpec(executable, []string{"--scenario=" + string(scenario)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return spec
}
