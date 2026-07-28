package sse

import (
	"testing"

	"mcp-gateway/internal/project"
)

func TestRegistryCapacityAndQuiesce(t *testing.T) {
	t.Parallel()
	registry := newRegistry()
	for index := 0; index < maxSessions; index++ {
		var id SessionID
		id[0] = byte(index)
		if _, ok := registry.reserve(id, projectNone()); !ok {
			t.Fatalf("se rechazó la sesión %d antes del límite", index)
		}
	}
	var extra SessionID
	extra[1] = 1
	if _, ok := registry.reserve(extra, projectNone()); ok {
		t.Fatal("se admitió una sesión sobre el límite")
	}
	registry.release(SessionID{})
	if _, ok := registry.reserve(extra, projectNone()); !ok {
		t.Fatal("no se liberó capacidad")
	}
	registry.quiesce()
	var final SessionID
	final[2] = 1
	if _, ok := registry.reserve(final, projectNone()); ok {
		t.Fatal("se admitió sesión durante quiesce")
	}
}

func projectNone() project.OptionalDir { return project.OptionalDir{} }
