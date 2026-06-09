// Package configfile handles atomic export and import of the shared
// config TOML mirror. Export writes providers/profiles from the store;
// import reads a TOML file and upserts records inside a transaction.
package configfile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/variableway/innate-aiswitcher/internal/safefile"
	"github.com/variableway/innate-aiswitcher/internal/store"
)

type Config struct {
	Version   int              `toml:"version"`
	Providers []store.Provider `toml:"providers"`
	Profiles  []store.Profile  `toml:"profiles"`
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "config.toml"
	}
	return filepath.Join(home, ".innate-aiswitcher", "config.toml")
}

func InitConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "init-config.toml"
	}
	return filepath.Join(home, ".innate-aiswitcher", "init-config.toml")
}

func DefaultBackupPath() string {
	base := filepath.Join(".", "backups")
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		base = filepath.Join(home, ".innate-aiswitcher", "backups")
	}
	name := "config-" + time.Now().Format("20060102-150405") + ".toml"
	return filepath.Join(base, name)
}

func Backup(s *store.Store, path string) (string, error) {
	if path == "" {
		path = DefaultBackupPath()
	}
	return path, Export(s, path, true)
}

func Dump(s *store.Store, path string) error {
	return Export(s, path, true)
}

func Export(s *store.Store, path string, includeSecrets bool) error {
	providers, err := s.ListProviders()
	if err != nil {
		return err
	}
	if !includeSecrets {
		for i := range providers {
			providers[i].APIKey = ""
		}
	}
	profiles, err := s.ListProfiles()
	if err != nil {
		return err
	}
	config := Config{Version: 1, Providers: providers, Profiles: profiles}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(config); err != nil {
		return err
	}
	return safefile.Write(path, buf.Bytes(), 0o600)
}

func Import(s *store.Store, path string) error {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return err
	}
	if config.Version == 0 {
		return fmt.Errorf("missing config version")
	}
	return s.RunInTransaction(func(tx *store.Store) error {
		for _, provider := range config.Providers {
			if _, err := tx.UpsertProvider(provider); err != nil {
				return fmt.Errorf("import provider %s: %w", provider.Slug, err)
			}
		}
		for _, profile := range config.Profiles {
			if _, err := tx.UpsertProfile(profile); err != nil {
				return fmt.Errorf("import profile %s: %w", profile.Slug, err)
			}
		}
		return nil
	})
}
