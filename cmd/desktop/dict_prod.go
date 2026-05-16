//go:build production

package main

import (
	"os"
	"path/filepath"
)

func resolveDictPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "dict.json"
	}
	path := filepath.Join(dir, "Morpho")
	_ = os.MkdirAll(path, 0o755)
	return filepath.Join(path, "dict.json")
}
