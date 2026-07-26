//go:build windows

package proc

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func validateExecutable(path string, info fs.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("el ejecutable no es un archivo regular")
	}
	extension := strings.ToUpper(filepath.Ext(path))
	pathExt := os.Getenv("PATHEXT")
	if pathExt == "" {
		pathExt = ".COM;.EXE;.BAT;.CMD"
	}
	for _, allowed := range strings.Split(pathExt, ";") {
		if extension == strings.ToUpper(strings.TrimSpace(allowed)) {
			return nil
		}
	}
	return fmt.Errorf("la extensión no está admitida por PATHEXT")
}
