//go:build !windows

package proc

import (
	"fmt"
	"io/fs"

	"golang.org/x/sys/unix"
)

func validateExecutable(path string, info fs.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("el ejecutable no es un archivo regular")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("el archivo no tiene bits de ejecución")
	}
	if err := unix.Access(path, unix.X_OK); err != nil {
		return fmt.Errorf("el archivo no es ejecutable por el usuario actual: %w", err)
	}
	return nil
}
