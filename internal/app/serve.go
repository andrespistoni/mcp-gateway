package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/proxy"
	"mcp-gateway/internal/sse"
)

type servingProxy interface {
	startupProxy
	Catalog() proxy.CatalogSnapshot
}

// Serve composes the listener before downstream startup and keeps the HTTP
// package independent from proxy and app. Shutdown coordination is completed
// in S-6; this slice closes acquired resources on context cancellation.
func (r *Runtime) Serve(ctx context.Context, requestedPort *endpoint.Port) error {
	if r.startProxy == nil {
		return fmt.Errorf("composición de startup incompleta")
	}
	snapshot, err := r.config.Load(ctx)
	if err != nil {
		return err
	}
	configured := snapshot.Port()
	port := endpoint.ResolvePort(requestedPort, &configured)
	signalCtx, stopSignals := runtimeSignalContext(ctx)
	defer stopSignals()
	server, err := sse.Bind(signalCtx, port, sse.Dependencies{})
	if err != nil {
		return err
	}
	service, err := r.startProxy(signalCtx, snapshot.Downstreams())
	if err != nil {
		_ = server.Close()
		return err
	}
	proxyService, ok := service.(servingProxy)
	if !ok {
		_ = service.Shutdown(signalCtx)
		_ = server.Close()
		return fmt.Errorf("proxy no expone catálogo")
	}
	caller, ok := service.(sse.Caller)
	if !ok || server.SetCaller(caller) != nil {
		_ = service.Shutdown(signalCtx)
		_ = server.Close()
		return fmt.Errorf("proxy no expone caller")
	}
	if err := server.SetCatalog(proxyService.Catalog()); err != nil {
		_ = service.Shutdown(signalCtx)
		_ = server.Close()
		return err
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve() }()
	select {
	case err := <-serveDone:
		_ = shutdownRuntime(server, service)
		return err
	case <-signalCtx.Done():
		_ = shutdownRuntime(server, service)
		<-serveDone
		return signalCtx.Err()
	}
}

type shutdownServer interface {
	Quiesce()
	Shutdown(context.Context) error
	Close() error
}

func shutdownRuntime(server shutdownServer, service startupProxy) error {
	deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Quiesce()
	var wait sync.WaitGroup
	var serverErr, proxyErr error
	wait.Add(2)
	go func() { defer wait.Done(); serverErr = server.Shutdown(deadline) }()
	go func() { defer wait.Done(); proxyErr = service.Shutdown(deadline) }()
	done := make(chan struct{})
	go func() { wait.Wait(); close(done) }()
	select {
	case <-done:
		if serverErr != nil {
			return serverErr
		}
		return proxyErr
	case <-deadline.Done():
		_ = server.Close()
		service.Kill()
		service.Wait()
		return deadline.Err()
	}
}
