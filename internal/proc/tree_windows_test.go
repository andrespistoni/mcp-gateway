//go:build windows

package proc

import (
	"strings"
	"testing"
)

func TestWindowsCommandLinePreservesUnicodeAndSpaces(t *testing.T) {
	line := buildCommandLine(`C:\Program Files\mcp.exe`, []string{"a b", "ñ", `x\"y`})
	for _, expected := range []string{`"C:\Program Files\mcp.exe"`, `"a b"`, "ñ"} {
		if !strings.Contains(line, expected) {
			t.Fatalf("command line %q no contiene %q", line, expected)
		}
	}
}

func TestWindowsAttributeConstantsAreAtomicCreationAttributes(t *testing.T) {
	if procThreadAttributeHandleList != 0x00020002 || procThreadAttributeJobList != 0x0002000d {
		t.Fatal("atributos STARTUPINFOEX inesperados")
	}
}
