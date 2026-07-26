//go:build !windows

package proc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestTreeTerminatesProcessGroupAndWaitIsStable(t *testing.T) {
	if os.Getenv("MCP_GATEWAY_TREE_HELPER") == "1" {
		child := exec.Command(os.Args[0], "-test.run=TestTreeGrandchildHelper")
		child.Env = append(os.Environ(), "MCP_GATEWAY_GRANDCHILD=1")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		_, _ = os.Stdout.WriteString(strconv.Itoa(child.Process.Pid) + "\n")
		select {}
	}
	executable, err := ResolveExecutable(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	spec, err := NewExecSpec(executable, []string{"-test.run=TestTreeTerminatesProcessGroupAndWaitIsStable"}, map[string]string{"MCP_GATEWAY_TREE_HELPER": "1"})
	if err != nil {
		t.Fatal(err)
	}
	tree, err := Start(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Stdin().Close()
	defer tree.Stdout().Close()
	defer tree.Stderr().Close()
	var pid int
	if _, err := fmt.Fscan(tree.Stdout(), &pid); err != nil {
		t.Fatal(err)
	}
	if err := tree.Terminate(); err != nil {
		t.Fatal(err)
	}
	_ = tree.Wait()
	first := tree.Wait()
	second := tree.Wait()
	if (first == nil) != (second == nil) {
		t.Fatalf("Wait no es estable: %v / %v", first, second)
	}
	deadline := time.Now().Add(2 * time.Second)
	for processAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if processAlive(pid) {
		t.Fatalf("el hijo %d sobrevivió al grupo", pid)
	}
}

func TestTreeDrainsLargeOutputAfterImmediateExit(t *testing.T) {
	const size = 768 * 1024
	if os.Getenv("MCP_GATEWAY_DRAIN_HELPER") == "1" {
		payload := bytes.Repeat([]byte("x"), size)
		if _, err := os.Stdout.Write(payload); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
	executable, err := ResolveExecutable(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	spec, err := NewExecSpec(executable, []string{"-test.run=TestTreeDrainsLargeOutputAfterImmediateExit"}, map[string]string{"MCP_GATEWAY_DRAIN_HELPER": "1"})
	if err != nil {
		t.Fatal(err)
	}
	tree, err := Start(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Stdin().Close()
	defer tree.Stdout().Close()
	defer tree.Stderr().Close()
	data, readErr := io.ReadAll(tree.Stdout())
	waitErr := tree.Wait()
	if readErr != nil || waitErr != nil || len(data) != size {
		t.Fatalf("drenado=%d read=%v wait=%v", len(data), readErr, waitErr)
	}
}

func TestTreeGrandchildHelper(t *testing.T) {
	if os.Getenv("MCP_GATEWAY_GRANDCHILD") != "1" {
		return
	}
	select {}
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
