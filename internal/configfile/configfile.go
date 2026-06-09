// Package configfile handles atomic export and import of the shared
// config mirror. Export writes providers/profiles from the store; import
// reads either a TOML or JSON file and upserts records inside a transaction.
package configfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/variableway/innate-aiswitcher/internal/safefile"
	"github.com/variableway/innate-aiswitcher/internal/store"
)

// Format selects the wire encoding for export/import.
type Format string

const (
	FormatTOML Format = "toml"
	FormatJSON Format = "json"
)

func (f Format) Valid() bool {
	switch f {
	case FormatTOML, FormatJSON:
		return true
	}
	return false
}

func (f Format) Extension() string {
	switch f {
	case FormatJSON:
		return ".json"
	case FormatTOML:
		return ".toml"
	}
	return ""
}

type Config struct {
	Version   int              `json:"version" toml:"version"`
	Providers []store.Provider `json:"providers" toml:"providers"`
	Profiles  []store.Profile  `json:"profiles" toml:"profiles"`
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
	return path, Export(s, path, FormatTOML, true)
}

func Dump(s *store.Store, path string) error {
	return Export(s, path, FormatTOML, true)
}

func DumpWithFormat(s *store.Store, path string, format Format) error {
	return Export(s, path, format, true)
}

// DetectFormat infers the wire format from a path's extension, falling
// back to a content sniff (first non-whitespace byte) when the extension
// is unknown. JSON starts with `{`; anything else (`#` comment, `[section]`,
// a bare key) is treated as TOML.
func DetectFormat(path string) (Format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return FormatJSON, nil
	case ".toml":
		return FormatTOML, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return FormatJSON, nil
		default:
			return FormatTOML, nil
		}
	}
	return "", fmt.Errorf("cannot detect format of %s (empty file)", path)
}

func Export(s *store.Store, path string, format Format, includeSecrets bool) error {
	if format == "" {
		format = FormatTOML
	}
	if !format.Valid() {
		return fmt.Errorf("unsupported export format %q", format)
	}
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
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(config); err != nil {
			return err
		}
	case FormatTOML:
		if err := toml.NewEncoder(&buf).Encode(config); err != nil {
			return err
		}
	}
	return safefile.Write(path, buf.Bytes(), 0o600)
}

func Import(s *store.Store, path string, format Format) error {
	if format == "" {
		detected, err := DetectFormat(path)
		if err != nil {
			return err
		}
		format = detected
	}
	if !format.Valid() {
		return fmt.Errorf("unsupported import format %q", format)
	}

	var config Config
	switch format {
	case FormatJSON:
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&config); err != nil {
			return fmt.Errorf("decode json: %w", err)
		}
	case FormatTOML:
		if _, err := toml.DecodeFile(path, &config); err != nil {
			return err
		}
	}
	if config.Version == 0 {
		return errors.New("missing config version")
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
