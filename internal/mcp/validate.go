package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func DecodeObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil, fmt.Errorf("se esperaba un objeto JSON")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, fmt.Errorf("se esperaba un objeto JSON")
	}
	return cloneFields(fields), nil
}
