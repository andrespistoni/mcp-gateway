//go:build !windows

package proxy

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"mcp-gateway/internal/proc"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestProbeCleansChildAndGrandchild(t *testing.T) {
	binary := buildFake(t)
	marker := t.TempDir() + "/pids"
	executable, err := proc.ResolveExecutable(binary)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := proc.NewExecSpec(executable, []string{"--scenario=" + string(fakemcp.ProcessTree), "--marker=" + marker}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Probe(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(marker)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var pids []int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		_, raw, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			t.Fatalf("marker inválido: %q", scanner.Text())
		}
		pid, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatal(err)
		}
		pids = append(pids, pid)
	}
	if len(pids) != 2 {
		t.Fatalf("PIDs = %#v", pids)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		alive := false
		for _, pid := range pids {
			err := syscall.Kill(pid, 0)
			alive = alive || err == nil || err == syscall.EPERM
		}
		if !alive {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("sobrevive algún descendiente: %#v", pids)
}
