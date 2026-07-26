package config

import "mcp-gateway/internal/endpoint"

const SchemaVersion = 1

type Downstream struct {
	Name            string            `yaml:"name"`
	Prefix          string            `yaml:"prefix"`
	Binary          string            `yaml:"binary"`
	Args            []string          `yaml:"args"`
	Enabled         bool              `yaml:"enabled"`
	Env             map[string]string `yaml:"env"`
	InjectProject   bool              `yaml:"inject_project"`
	ProjectArgument string            `yaml:"project_argument,omitempty"`
}

type Document struct {
	Version     int           `yaml:"version"`
	Port        endpoint.Port `yaml:"port"`
	Downstreams []Downstream  `yaml:"downstreams"`
}

func NewDocument() Document {
	return Document{
		Version:     SchemaVersion,
		Port:        endpoint.MustPort(endpoint.DefaultPort),
		Downstreams: []Downstream{},
	}
}

func (d Document) Clone() Document {
	clone := d
	clone.Downstreams = make([]Downstream, len(d.Downstreams))
	for i, downstream := range d.Downstreams {
		clone.Downstreams[i] = downstream
		clone.Downstreams[i].Args = make([]string, len(downstream.Args))
		copy(clone.Downstreams[i].Args, downstream.Args)
		clone.Downstreams[i].Env = make(map[string]string, len(downstream.Env))
		for key, value := range downstream.Env {
			clone.Downstreams[i].Env[key] = value
		}
	}
	return clone
}

type Snapshot struct {
	document Document
}

func newSnapshot(document Document) Snapshot {
	return Snapshot{document: document.Clone()}
}

func (s Snapshot) Document() Document {
	return s.document.Clone()
}

func (s Snapshot) Port() endpoint.Port {
	return s.document.Port
}

func (s Snapshot) Downstreams() []Downstream {
	return s.document.Clone().Downstreams
}
