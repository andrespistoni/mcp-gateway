package e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGitHubWorkflowsAreValidYAML(t *testing.T) {
	for _, name := range []string{"quality.yml", "release.yml"} {
		path := filepath.Join(repositoryRoot(t), ".github", "workflows", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var document yaml.Node
		if err := yaml.Unmarshal(data, &document); err != nil {
			t.Fatalf("%s no es YAML válido: %v", name, err)
		}
		if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
			t.Fatalf("%s no contiene un documento de workflow", name)
		}
	}
}
