package app

import (
	"context"
	"testing"

	"mcp-gateway/internal/config"
)

func TestRemoveAndSetEnabledSemantics(t *testing.T) {
	repository := testRepository(t)
	_, err := repository.Update(context.Background(), func(document *config.Document) error {
		document.Downstreams = []config.Downstream{{
			Name: "one", Prefix: "one__", Binary: "one", Args: []string{}, Enabled: true, Env: map[string]string{},
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runtimeApp := NewRuntime(repository)
	commits := 0
	runtimeApp.mutationCommitted = func(context.Context) error {
		commits++
		return nil
	}
	changed, err := runtimeApp.SetEnabled(context.Background(), "one", false)
	if err != nil || !changed {
		t.Fatalf("disable = %v, %v", changed, err)
	}
	changed, err = runtimeApp.SetEnabled(context.Background(), "one", false)
	if err != nil || changed {
		t.Fatalf("disable idempotente = %v, %v", changed, err)
	}
	if commits != 1 {
		t.Fatalf("seam invocado %d veces antes de remove", commits)
	}
	if _, err := runtimeApp.SetEnabled(context.Background(), "missing", true); err == nil {
		t.Fatal("enable ausente debía fallar")
	}
	changed, err = runtimeApp.Remove(context.Background(), "one")
	if err != nil || !changed {
		t.Fatalf("remove = %v, %v", changed, err)
	}
	if commits != 2 {
		t.Fatalf("seam invocado %d veces", commits)
	}
	if _, err := runtimeApp.Remove(context.Background(), "one"); err == nil {
		t.Fatal("remove ausente debía fallar")
	}
}
