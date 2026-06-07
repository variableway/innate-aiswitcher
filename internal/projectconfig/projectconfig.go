// Package projectconfig discovers per-directory .aiswrc configuration files.
// It walks up from the current directory looking for .aiswrc and parses the
// profile/agent/provider binding stored there.
package projectconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ProjectConfig is the on-disk format for .aiswrc
type ProjectConfig struct {
	Profile  string `toml:"profile"`
	Agent    string `toml:"agent"`
	Provider string `toml:"provider"`
}

// Find walks up from startDir (or cwd if empty) looking for .aiswrc.
// Returns the parsed config and the directory it was found in.
func Find(startDir string) (*ProjectConfig, string, error) {
	dir := startDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, "", err
		}
	}

	for {
		path := filepath.Join(dir, ".aiswrc")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			cfg, err := Load(path)
			if err != nil {
				return nil, "", fmt.Errorf("parse %s: %w", path, err)
			}
			return cfg, dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, "", nil
}

// Load parses a single .aiswrc file.
func Load(path string) (*ProjectConfig, error) {
	var cfg ProjectConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	cfg.Profile = strings.TrimSpace(cfg.Profile)
	cfg.Agent = strings.TrimSpace(cfg.Agent)
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	return &cfg, nil
}

// Write creates or overwrites .aiswrc in the given directory.
func Write(dir string, cfg ProjectConfig) error {
	path := filepath.Join(dir, ".aiswrc")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if cfg.Profile != "" {
		fmt.Fprintf(f, "profile = %q\n", cfg.Profile)
	}
	if cfg.Agent != "" {
		fmt.Fprintf(f, "agent = %q\n", cfg.Agent)
	}
	if cfg.Provider != "" {
		fmt.Fprintf(f, "provider = %q\n", cfg.Provider)
	}
	return f.Sync()
}
