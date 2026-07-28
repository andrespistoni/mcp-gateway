package sse

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"

	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/project"
)

// Catalog is intentionally declared by its consumer. proxy.CatalogSnapshot
// satisfies it without importing this package.
type Catalog interface {
	Tools() []mcp.Tool
	Identity() [32]byte
}

// Caller is declared by the SSE consumer. Implementations reserve capacity
// synchronously; false means no work was admitted and maps to HTTP 429.
type Caller interface {
	TryCall(context.Context, mcp.RawID, json.RawMessage, project.OptionalDir) (<-chan mcp.Envelope, func(), bool)
}

type Dependencies struct {
	Catalog Catalog
	Caller  Caller
	Entropy io.Reader
	Version string
}

func (d Dependencies) entropy() io.Reader {
	if d.Entropy == nil {
		return rand.Reader
	}
	return d.Entropy
}
