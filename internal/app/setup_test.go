package app

import (
	"context"
	"os"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/discovery"
	"mcp-gateway/internal/endpoint"
)

func TestSetupConvergesConfigDiscoveryAndDaemonWithFakes(t *testing.T) {
	repository := testRepository(t)
	manager := &fakeDaemonManager{}
	runtime := &Runtime{
		config: repository,
		discovery: staticDiscovery{result: discovery.Result{Downstreams: []config.Downstream{{
			Name: "engram", Prefix: "engram__", Binary: "/temporary/engram", Args: []string{}, Enabled: true, Env: map[string]string{},
		}}}},
		daemonManager: manager,
	}
	port := endpoint.MustPort(4444)
	first, err := runtime.Setup(context.Background(), SetupRequest{Port: &port})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(repository.Path())
	if err != nil {
		t.Fatal(err)
	}
	second, err := runtime.Setup(context.Background(), SetupRequest{Port: &port})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(repository.Path())
	if err != nil {
		t.Fatal(err)
	}
	if first.Port.Number() != 4444 || !first.Discovery.Items[0].Added || second.Discovery.Items[0].Added || string(before) != string(after) {
		t.Fatalf("first=%#v second=%#v config changed=%v", first, second, string(before) != string(after))
	}
	if got := len(manager.calls); got != 2 || manager.calls[0] != "enable" || manager.calls[1] != "enable" {
		t.Fatalf("daemon calls=%#v", manager.calls)
	}
}
