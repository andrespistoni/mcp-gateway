package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/proc"
)

type processStarter func(proc.ExecSpec) (proc.ProcessTree, error)

type startupResult struct {
	tools []mcp.Tool
	err   error
}

type managedProcess struct {
	tree          proc.ProcessTree
	requests      chan writeRequest
	inbound       chan readResult
	calls         chan *call
	cancellations chan *call
	credits       chan struct{}
	stop          chan struct{}
	actorGone     chan struct{}
	done          chan struct{}
	waitDone      chan struct{}
	stopOnce      sync.Once
	ioWG          sync.WaitGroup
	stderr        stderrCapture
	onFailure     func(error)
	intentMu      sync.Mutex
	intentional   bool
}

func startManagedProcess(ctx context.Context, spec proc.ExecSpec, starter processStarter, onFailure func(error)) (*managedProcess, []mcp.Tool, error) {
	tree, err := starter(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo iniciar downstream MCP: %w", err)
	}
	process := &managedProcess{
		tree: tree, requests: make(chan writeRequest), inbound: make(chan readResult, 1),
		calls: make(chan *call, maxQueuedCalls), cancellations: make(chan *call, maxQueuedCalls+1),
		credits: make(chan struct{}, maxQueuedCalls),
		stop:    make(chan struct{}), actorGone: make(chan struct{}), done: make(chan struct{}),
		waitDone: make(chan struct{}), onFailure: onFailure,
	}
	for range maxQueuedCalls {
		process.credits <- struct{}{}
	}
	startup := make(chan startupResult, 1)
	go process.runIO()
	go process.runActor(ctx, startup)
	result := <-startup
	if result.err != nil {
		<-process.done
		return nil, nil, result.err
	}
	return process, result.tools, nil
}

func (p *managedProcess) runIO() {
	go func() {
		_ = p.tree.Wait()
		close(p.waitDone)
	}()
	p.ioWG.Add(3)
	go func() {
		defer p.ioWG.Done()
		writeStdin(p.tree.Stdin(), p.requests)
	}()
	go func() {
		defer p.ioWG.Done()
		readStdout(p.tree.Stdout(), p.inbound, p.actorGone)
	}()
	go func() {
		defer p.ioWG.Done()
		stderr := p.tree.Stderr()
		defer stderr.Close()
		p.stderr.drain(stderr)
	}()
}

func (p *managedProcess) stopAndWait(ctx context.Context) {
	p.intentMu.Lock()
	p.intentional = true
	p.intentMu.Unlock()
	p.stopOnce.Do(func() { close(p.stop) })
	select {
	case <-p.done:
	case <-ctx.Done():
		_ = p.tree.Kill()
		<-p.done
	}
}

func (p *managedProcess) kill() { _ = p.tree.Kill() }

func (p *managedProcess) isIntentional() bool {
	p.intentMu.Lock()
	defer p.intentMu.Unlock()
	return p.intentional
}

func (p *managedProcess) cleanup() {
	close(p.actorGone)
	close(p.requests)
	_ = p.tree.Terminate()
	select {
	case <-p.waitDone:
	case <-time.After(time.Second):
		_ = p.tree.Kill()
		<-p.waitDone
	}
	p.ioWG.Wait()
	// Reader and stderr owners have drained their parent handles to EOF and
	// closed them themselves; cleanup never closes those handles on their behalf.
}
