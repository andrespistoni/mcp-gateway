package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeAppliesDefaultsAndPreservesRawReferences(t *testing.T) {
	data := []byte(`version: 1
downstreams:
  - name: example
    prefix: example__
    binary: example-mcp
    env:
      API_TOKEN: "${MISSING_TOKEN}"
    inject_project: true
  - name: disabled
    prefix: disabled__
    binary: disabled-mcp
    enabled: false
`)
	snapshot, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Port().Number() != 3333 {
		t.Fatalf("port = %d", snapshot.Port().Number())
	}
	downstreams := snapshot.Downstreams()
	if !downstreams[0].Enabled || downstreams[0].ProjectArgument != "projectPath" || downstreams[0].Env["API_TOKEN"] != "${MISSING_TOKEN}" {
		t.Fatalf("defaults/referencia = %#v", downstreams[0])
	}
	if downstreams[1].Enabled {
		t.Fatal("enabled:false no se preservó")
	}
	if downstreams[0].Args == nil || downstreams[0].Env == nil {
		t.Fatal("colecciones por defecto deben ser no nil")
	}
}

func TestDecodeRejectsStrictYAMLAndInvalidRules(t *testing.T) {
	tests := map[string]string{
		"vacío":             "",
		"version ausente":   "downstreams: []\n",
		"version":           "version: 2\n",
		"campo desconocido": "version: 1\nextra: true\n",
		"clave duplicada":   "version: 1\nversion: 1\n",
		"anchor":            "version: &v 1\nport: *v\n",
		"trailing":          "version: 1\n---\nversion: 1\n",
		"puerto":            "version: 1\nport: 80\n",
		"nombre":            "version: 1\ndownstreams:\n- name: 'á'\n  prefix: a__\n  binary: a\n",
		"nombre duplicado":  "version: 1\ndownstreams:\n- {name: a, prefix: a__, binary: a}\n- {name: a, prefix: b__, binary: b}\n",
		"prefix duplicado":  "version: 1\ndownstreams:\n- {name: a, prefix: a__, binary: a}\n- {name: b, prefix: a__, binary: b}\n",
		"prefix triple":     "version: 1\ndownstreams:\n- {name: a, prefix: a___, binary: a}\n",
		"env":               "version: 1\ndownstreams:\n- name: a\n  prefix: a__\n  binary: a\n  env: {'BAD-KEY': value}\n",
		"project argument":  "version: 1\ndownstreams:\n- name: a\n  prefix: a__\n  binary: a\n  inject_project: true\n  project_argument: \"bad\\x01\"\n",
		"project vacío":     "version: 1\ndownstreams:\n- name: a\n  prefix: a__\n  binary: a\n  inject_project: true\n  project_argument: ''\n",
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode([]byte(data)); err == nil {
				t.Fatalf("Decode debía rechazar:\n%s", data)
			}
		})
	}
}

func TestSnapshotIsDefensive(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "valid.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	first := snapshot.Downstreams()
	first[0].Name = "alterado"
	first[0].Args = append(first[0].Args, "x")
	if got := snapshot.Downstreams()[0]; got.Name != "ejemplo" || len(got.Args) != 0 {
		t.Fatalf("snapshot mutado: %#v", got)
	}
}

func TestValidationRejectsNUL(t *testing.T) {
	for _, field := range []string{"binary", "arg", "env"} {
		document := NewDocument()
		downstream := Downstream{Name: "a", Prefix: "a__", Binary: "a", Args: []string{}, Env: map[string]string{}}
		switch field {
		case "binary":
			downstream.Binary = "a\x00b"
		case "arg":
			downstream.Args = []string{"a\x00b"}
		case "env":
			downstream.Env["VALUE"] = "a\x00b"
		}
		document.Downstreams = []Downstream{downstream}
		if err := Validate(&document); err == nil || !strings.Contains(err.Error(), "inválid") && !strings.Contains(err.Error(), "NUL") {
			t.Fatalf("%s no rechazado: %v", field, err)
		}
	}
}
