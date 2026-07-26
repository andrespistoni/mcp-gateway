package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/proxy"
)

type startupConfig struct {
	snapshot config.Snapshot
	err      error
}

func (c startupConfig) Load(context.Context) (config.Snapshot, error) { return c.snapshot, c.err }

type recordingListener struct {
	events *[]string
}

func (l *recordingListener) Close() error {
	*l.events = append(*l.events, "listener-close")
	return nil
}

type recordingProxy struct {
	events *[]string
}

func (p *recordingProxy) Shutdown(ctx context.Context) error {
	*p.events = append(*p.events, "proxy-shutdown")
	return ctx.Err()
}
func (p *recordingProxy) Kill() { *p.events = append(*p.events, "proxy-kill") }
func (p *recordingProxy) Wait() { *p.events = append(*p.events, "proxy-wait") }

func TestStartupAcquisitionOrderAndCancelledRollback(t *testing.T) {
	snapshot, err := config.Decode([]byte("version: 1\nport: 4444\ndownstreams: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(startupConfig{snapshot: snapshot})
	events := []string{}
	runtime.startProxy = func(context.Context, []config.Downstream) (startupProxy, error) {
		events = append(events, "proxy-start")
		return &recordingProxy{events: &events}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	requested := endpoint.MustPort(5555)
	resources, port, err := runtime.startGateway(ctx, &requested, func(_ context.Context, acquiredPort endpoint.Port) (startupListener, error) {
		if acquiredPort.Number() != requested.Number() {
			t.Fatalf("listener recibió puerto %d", acquiredPort.Number())
		}
		events = append(events, "listener-acquire")
		return &recordingListener{events: &events}, nil
	}, func(startupListener, startupProxy) error {
		events = append(events, "compose")
		return nil
	})
	if err != nil || port.Number() != 5555 {
		t.Fatalf("startup port=%d err=%v", port.Number(), err)
	}
	cancel()
	if err := resources.Close(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("close = %v", err)
	}
	want := "[listener-acquire proxy-start compose proxy-shutdown proxy-kill proxy-wait listener-close]"
	if fmt.Sprint(events) != want {
		t.Fatalf("orden = %v", events)
	}
	_ = resources.Close(context.Background())
	if fmt.Sprint(events) != want {
		t.Fatal("rollback no fue idempotente")
	}
}

func TestStartupFailuresRollbackOnlyAcquiredResources(t *testing.T) {
	snapshot, err := config.Decode([]byte("version: 1\ndownstreams: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("listener", func(t *testing.T) {
		runtime := NewRuntime(startupConfig{snapshot: snapshot})
		started := false
		runtime.startProxy = func(context.Context, []config.Downstream) (startupProxy, error) {
			started = true
			return nil, nil
		}
		_, _, err := runtime.startGateway(context.Background(), nil, func(context.Context, endpoint.Port) (startupListener, error) {
			return nil, errors.New("listener")
		}, func(startupListener, startupProxy) error { return nil })
		if err == nil || started {
			t.Fatalf("err=%v proxy-started=%v", err, started)
		}
	})
	t.Run("proxy", func(t *testing.T) {
		events := []string{}
		runtime := NewRuntime(startupConfig{snapshot: snapshot})
		runtime.startProxy = func(context.Context, []config.Downstream) (startupProxy, error) {
			return nil, errors.New("collision")
		}
		_, _, err := runtime.startGateway(context.Background(), nil, func(context.Context, endpoint.Port) (startupListener, error) {
			return &recordingListener{events: &events}, nil
		}, func(startupListener, startupProxy) error { return nil })
		if err == nil || fmt.Sprint(events) != "[listener-close]" {
			t.Fatalf("err=%v events=%v", err, events)
		}
	})
}

func TestStartupWithZeroDownstreamsUsesRealProxy(t *testing.T) {
	snapshot, err := config.Decode([]byte("version: 1\nport: 4444\ndownstreams: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(startupConfig{snapshot: snapshot})
	events := []string{}
	resources, port, err := runtime.startGateway(context.Background(), nil, func(context.Context, endpoint.Port) (startupListener, error) {
		return &recordingListener{events: &events}, nil
	}, func(_ startupListener, service startupProxy) error {
		concrete, ok := service.(*proxy.Service)
		if !ok || len(concrete.Catalog().Tools()) != 0 {
			t.Fatalf("proxy real/catálogo vacío inválido: %T", service)
		}
		return nil
	})
	if err != nil || port.Number() != 4444 {
		t.Fatalf("port=%d err=%v", port.Number(), err)
	}
	if err := resources.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(events) != "[listener-close]" {
		t.Fatalf("events=%v", events)
	}
}
