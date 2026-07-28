package app

import (
	"context"
	"errors"
	"testing"

	"mcp-gateway/internal/config"
)

type failingConfigLoader struct{ err error }

func (f failingConfigLoader) Load(context.Context) (config.Snapshot, error) {
	return config.Snapshot{}, f.err
}

func TestListPropagatesConfigurationFailure(t *testing.T) {
	want := errors.New("load failed")
	runtime := &Runtime{config: failingConfigLoader{err: want}}
	if _, err := runtime.List(context.Background()); !errors.Is(err, want) {
		t.Fatalf("List error = %v", err)
	}
}
