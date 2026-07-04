package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DictPath string `json:"dict_path"`
}

// resolveConfigPath returns the path to config.json located in the same directory as dict.json.
func resolveConfigPath() string {
	defaultDict := resolveDictPath()
	dir := filepath.Dir(defaultDict)
	return filepath.Join(dir, "config.json")
}

// loadConfig loads custom settings from config.json. Returns default settings if file doesn't exist.
func loadConfig() Config {
	path := resolveConfigPath()
	file, err := os.Open(path)
	if err != nil {
		return Config{DictPath: resolveDictPath()}
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{DictPath: resolveDictPath()}
	}
	if cfg.DictPath == "" {
		cfg.DictPath = resolveDictPath()
	}
	return cfg
}

// saveConfig saves the custom settings to config.json.
func saveConfig(cfg Config) error {
	path := resolveConfigPath()
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
