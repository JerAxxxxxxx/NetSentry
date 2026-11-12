// Package storage provides file writes that preserve the previous snapshot
// if preparing a replacement fails.
package storage

import (
	"os"
	"path/filepath"
)

func WriteFile(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".netsentry-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
