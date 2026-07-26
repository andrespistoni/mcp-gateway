//go:build !windows

package persist

import (
	"os"
	"path/filepath"
)

func replacePath(source, destination string) error {
	return os.Rename(source, destination)
}

func syncDirectory(path string) error {
	directory, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func restrictFileToUser(string) error {
	return nil
}

func restrictDirectoryToUser(string) error {
	return nil
}
