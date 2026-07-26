package mcp

import (
	"encoding/json"
	"fmt"
)

type Tool struct {
	name   string
	fields map[string]json.RawMessage
}

func ParseTool(raw json.RawMessage) (Tool, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return Tool{}, fmt.Errorf("tool debe ser un objeto")
	}
	var name string
	if err := json.Unmarshal(fields["name"], &name); err != nil || name == "" {
		return Tool{}, fmt.Errorf("tool.name debe ser string no vacío")
	}
	return Tool{name: name, fields: cloneFields(fields)}, nil
}

func (t Tool) Name() string { return t.name }

func (t Tool) Fields() map[string]json.RawMessage { return cloneFields(t.fields) }

func (t Tool) WithName(name string) (Tool, error) {
	if name == "" {
		return Tool{}, fmt.Errorf("tool.name debe ser string no vacío")
	}
	clone := cloneFields(t.fields)
	encoded, _ := json.Marshal(name)
	clone["name"] = encoded
	return Tool{name: name, fields: clone}, nil
}

func (t Tool) MarshalJSON() ([]byte, error) {
	if t.name == "" || t.fields == nil {
		return nil, fmt.Errorf("tool inválida")
	}
	return json.Marshal(t.fields)
}

type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      map[string]any `json:"serverInfo"`
}
