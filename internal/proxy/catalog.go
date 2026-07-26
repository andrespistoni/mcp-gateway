package proxy

import (
	"crypto/rand"
	"fmt"
	"io"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/mcp"
)

const MaxCatalogTools = 10000

type Route struct {
	Downstream      string
	OriginalName    string
	InjectProject   bool
	ProjectArgument string
}

type CatalogSnapshot struct {
	tools    []mcp.Tool
	routes   map[string]Route
	identity [32]byte
}

type catalogEntry struct {
	config config.Downstream
	tools  []mcp.Tool
}

func buildCatalog(entries []catalogEntry, entropy io.Reader) (CatalogSnapshot, error) {
	if entropy == nil {
		entropy = rand.Reader
	}
	snapshot := CatalogSnapshot{routes: make(map[string]Route)}
	if _, err := io.ReadFull(entropy, snapshot.identity[:]); err != nil {
		return CatalogSnapshot{}, fmt.Errorf("no se pudo crear identidad del catálogo: %w", err)
	}
	for _, entry := range entries {
		for _, tool := range entry.tools {
			if len(snapshot.tools) == MaxCatalogTools {
				return CatalogSnapshot{}, fmt.Errorf("el catálogo supera 10000 tools")
			}
			finalName := entry.config.Prefix + tool.Name()
			if _, duplicate := snapshot.routes[finalName]; duplicate {
				return CatalogSnapshot{}, fmt.Errorf("nombre final de tool duplicado: %s", finalName)
			}
			prefixed, err := tool.WithName(finalName)
			if err != nil {
				return CatalogSnapshot{}, err
			}
			snapshot.tools = append(snapshot.tools, prefixed)
			snapshot.routes[finalName] = Route{
				Downstream: entry.config.Name, OriginalName: tool.Name(),
				InjectProject: entry.config.InjectProject, ProjectArgument: entry.config.ProjectArgument,
			}
		}
	}
	return snapshot, nil
}

func (s CatalogSnapshot) Tools() []mcp.Tool {
	tools := make([]mcp.Tool, len(s.tools))
	for index, tool := range s.tools {
		tools[index], _ = tool.WithName(tool.Name())
	}
	return tools
}

func (s CatalogSnapshot) Route(name string) (Route, bool) {
	route, exists := s.routes[name]
	return route, exists
}

func (s CatalogSnapshot) Identity() [32]byte { return s.identity }
