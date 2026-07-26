package claude

import (
	"encoding/json"
	"testing"
)

func FuzzProjectMerge(f *testing.F) {
	f.Add([]byte(`{"future":{"value":1},"mcpServers":{"other":{"command":"x"}}}`))
	f.Add([]byte(`{"mcpServers":{}}`))
	f.Add([]byte(`not-json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		merged, _, err := mergeProjectJSON(data, true, "http://localhost:3333/sse?projectDir=%2Ftmp")
		if err != nil {
			return
		}
		if !json.Valid(merged) || len(merged) == 0 || merged[len(merged)-1] != '\n' {
			t.Fatal("merge produjo JSON inválido")
		}
		var before, after map[string]json.RawMessage
		if json.Unmarshal(data, &before) != nil || json.Unmarshal(merged, &after) != nil {
			t.Fatal("merge aceptó una raíz no decodificable")
		}
		for key, value := range before {
			if key == "mcpServers" {
				continue
			}
			if !jsonSemanticallyEqual(value, after[key]) {
				t.Fatalf("se alteró la clave %q", key)
			}
		}
		var beforeServers, afterServers map[string]json.RawMessage
		_ = json.Unmarshal(before["mcpServers"], &beforeServers)
		_ = json.Unmarshal(after["mcpServers"], &afterServers)
		for key, value := range beforeServers {
			if key != projectServerName && !jsonSemanticallyEqual(value, afterServers[key]) {
				t.Fatalf("se alteró el servidor %q", key)
			}
		}
	})
}

func jsonSemanticallyEqual(left, right json.RawMessage) bool {
	var leftValue, rightValue any
	return json.Unmarshal(left, &leftValue) == nil && json.Unmarshal(right, &rightValue) == nil &&
		string(mustJSON(leftValue)) == string(mustJSON(rightValue))
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}
