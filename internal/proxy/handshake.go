package proxy

import (
	"encoding/json"
	"fmt"

	"mcp-gateway/internal/mcp"
)

func validateInitialize(envelope mcp.Envelope, expected mcp.RawID) error {
	if envelope.Kind() == mcp.Error {
		return fmt.Errorf("initialize devolvió un error JSON-RPC")
	}
	if envelope.Kind() != mcp.Result {
		return fmt.Errorf("initialize no devolvió un resultado")
	}
	id, ok := envelope.ID()
	if !ok || !id.Equal(expected) {
		return fmt.Errorf("initialize devolvió un id inesperado")
	}
	fields, err := mcp.DecodeObject(envelope.Result())
	if err != nil {
		return fmt.Errorf("resultado initialize inválido")
	}
	var version string
	if err := json.Unmarshal(fields["protocolVersion"], &version); err != nil || version != mcp.ProtocolVersion {
		return fmt.Errorf("versión MCP incompatible")
	}
	return nil
}
