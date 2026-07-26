package config

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type rawDocument struct {
	Version     *int            `yaml:"version"`
	Port        *int            `yaml:"port"`
	Downstreams []rawDownstream `yaml:"downstreams"`
}

type rawDownstream struct {
	Name            string             `yaml:"name"`
	Prefix          string             `yaml:"prefix"`
	Binary          string             `yaml:"binary"`
	Args            *[]string          `yaml:"args"`
	Enabled         *bool              `yaml:"enabled"`
	Env             *map[string]string `yaml:"env"`
	InjectProject   *bool              `yaml:"inject_project"`
	ProjectArgument *string            `yaml:"project_argument"`
}

func Decode(data []byte) (Snapshot, error) {
	if err := inspectYAML(data); err != nil {
		return Snapshot{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var raw rawDocument
	if err := decoder.Decode(&raw); err != nil {
		return Snapshot{}, fmt.Errorf("YAML inválido: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Snapshot{}, fmt.Errorf("solo se admite un documento YAML")
		}
		return Snapshot{}, fmt.Errorf("documento YAML adicional inválido: %w", err)
	}
	document, err := fromRaw(raw)
	if err != nil {
		return Snapshot{}, err
	}
	if err := Validate(&document); err != nil {
		return Snapshot{}, err
	}
	return newSnapshot(document), nil
}

func inspectYAML(data []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var node yaml.Node
	if err := decoder.Decode(&node); err != nil {
		return fmt.Errorf("YAML inválido: %w", err)
	}
	if len(node.Content) == 0 {
		return fmt.Errorf("el documento YAML está vacío")
	}
	if err := inspectNode(&node); err != nil {
		return err
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("solo se admite un documento YAML")
		}
		return fmt.Errorf("documento YAML adicional inválido: %w", err)
	}
	return nil
}

func inspectNode(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return fmt.Errorf("los aliases y anchors YAML no están permitidos")
	}
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]struct{}, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode {
				return fmt.Errorf("las claves YAML deben ser escalares")
			}
			if _, exists := seen[key.Value]; exists {
				return fmt.Errorf("clave YAML duplicada: %s", key.Value)
			}
			seen[key.Value] = struct{}{}
		}
	}
	for _, child := range node.Content {
		if err := inspectNode(child); err != nil {
			return err
		}
	}
	return nil
}
