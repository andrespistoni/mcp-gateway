package proxy

import (
	"context"
	"io"
	"sync/atomic"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/proc"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestProcessTransportJoinsOwnersAfterImmediateExit(t *testing.T) {
	binary := buildFake(t)
	service, err := Start(context.Background(), []config.Downstream{
		downstream("exits", "exits__", binary, fakemcp.Healthy, true),
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForState(t, service, "exits", StateUnavailable)
	service.Wait()
	if got := toolNames(service.Catalog()); len(got) != 1 || got[0] != "exits__echo" {
		t.Fatalf("catálogo tras EOF = %v", got)
	}
}

type countingTree struct {
	proc.ProcessTree
	stdinGets, stdoutGets, stderrGets, waits atomic.Int32
	stdinCloses, stdoutCloses, stderrCloses  atomic.Int32
}

type countingReadCloser struct {
	io.ReadCloser
	closes *atomic.Int32
}

func (r countingReadCloser) Close() error {
	r.closes.Add(1)
	return r.ReadCloser.Close()
}

type countingWriteCloser struct {
	io.WriteCloser
	closes *atomic.Int32
}

func (w countingWriteCloser) Close() error {
	w.closes.Add(1)
	return w.WriteCloser.Close()
}

func (t *countingTree) Stdin() io.WriteCloser {
	t.stdinGets.Add(1)
	return countingWriteCloser{WriteCloser: t.ProcessTree.Stdin(), closes: &t.stdinCloses}
}

func (t *countingTree) Stdout() io.ReadCloser {
	t.stdoutGets.Add(1)
	return countingReadCloser{ReadCloser: t.ProcessTree.Stdout(), closes: &t.stdoutCloses}
}

func (t *countingTree) Stderr() io.ReadCloser {
	t.stderrGets.Add(1)
	return countingReadCloser{ReadCloser: t.ProcessTree.Stderr(), closes: &t.stderrCloses}
}

func (t *countingTree) Wait() error {
	t.waits.Add(1)
	return t.ProcessTree.Wait()
}

func TestProcessTransportHasSinglePipeOwnersAndWaiter(t *testing.T) {
	binary := buildFake(t)
	configured := downstream("owned", "owned__", binary, fakemcp.RuntimeHealthy, true)
	var tree *countingTree
	service, err := startService(context.Background(), []config.Downstream{configured}, func(spec proc.ExecSpec) (proc.ProcessTree, error) {
		started, err := proc.Start(spec)
		if err != nil {
			return nil, err
		}
		tree = &countingTree{ProcessTree: started}
		return tree, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	shutdownService(t, service)
	counts := []int32{
		tree.stdinGets.Load(), tree.stdoutGets.Load(), tree.stderrGets.Load(), tree.waits.Load(),
		tree.stdinCloses.Load(), tree.stdoutCloses.Load(), tree.stderrCloses.Load(),
	}
	for index, count := range counts {
		if count != 1 {
			t.Fatalf("ownership count[%d]=%d, todos=%v", index, count, counts)
		}
	}
}
