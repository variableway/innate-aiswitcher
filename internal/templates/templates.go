// Package templates loads embedded configuration templates and provider
// presets bundled into the binary. It is used by the CLI and TUI to
// generate example configs and guide provider setup.
package templates

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/variableway/innate-aiswitcher/internal/safefile"
	"github.com/variableway/innate-aiswitcher/internal/store"
)

//go:embed files/*
var files embed.FS

type ProviderPreset struct {
	Slug       string      `toml:"slug"`
	Name       string      `toml:"name"`
	URLOptions []URLOption `toml:"url_options"`
}

type URLOption struct {
	Slug         string                 `toml:"slug"`
	Label        string                 `toml:"label"`
	BaseURL      string                 `toml:"base_url"`
	APIProtocol  string                 `toml:"api_protocol"`
	DefaultModel string                 `toml:"default_model"`
	Endpoints    map[string]string      `toml:"endpoints"`
	Capabilities map[string]interface{} `toml:"capabilities"`
}

type presetFile struct {
	Presets []ProviderPreset `toml:"presets"`
}

func ConfigExample() (string, error) {
	bytes, err := files.ReadFile("files/config.example.toml")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func WriteConfigExample(path string) error {
	content, err := ConfigExample()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return safefile.Write(path, []byte(content), 0o600)
}

func ProviderPresets() ([]ProviderPreset, error) {
	bytes, err := files.ReadFile("files/provider-presets.toml")
	if err != nil {
		return nil, err
	}
	var data presetFile
	if _, err := toml.DecodeReader(bytesReader(bytes), &data); err != nil {
		return nil, err
	}
	sort.Slice(data.Presets, func(i, j int) bool { return data.Presets[i].Slug < data.Presets[j].Slug })
	return data.Presets, nil
}

func FindPreset(slug string) (*ProviderPreset, error) {
	presets, err := ProviderPresets()
	if err != nil {
		return nil, err
	}
	for i := range presets {
		if presets[i].Slug == slug {
			return &presets[i], nil
		}
	}
	return nil, fmt.Errorf("provider preset not found: %s", slug)
}

func ProviderFromPreset(preset ProviderPreset, option URLOption, apiKey string) store.Provider {
	slug := preset.Slug
	name := preset.Name
	if len(preset.URLOptions) > 1 || option.Slug != preset.Slug {
		slug = preset.Slug + "-" + option.Slug
		name = preset.Name + " " + option.Label
	}
	return store.Provider{
		Slug:         slug,
		Name:         name,
		BaseURL:      option.BaseURL,
		APIKey:       apiKey,
		APIProtocol:  option.APIProtocol,
		DefaultModel: option.DefaultModel,
		Endpoints:    option.Endpoints,
		Capabilities: option.Capabilities,
		Active:       true,
	}
}

func PresetLabel(preset ProviderPreset) string {
	choices := make([]string, 0, len(preset.URLOptions))
	for _, option := range preset.URLOptions {
		choices = append(choices, option.Label)
	}
	if len(choices) == 0 {
		return preset.Name + " (no URL options)"
	}
	return preset.Name + " (" + strings.Join(choices, ", ") + ")"
}

func OptionLabel(option URLOption) string {
	return fmt.Sprintf("%s - %s - %s", option.Label, option.APIProtocol, option.BaseURL)
}

func bytesReader(value []byte) *bytes.Reader {
	return bytes.NewReader(value)
}
