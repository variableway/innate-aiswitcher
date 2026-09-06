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

// ProviderPreset is a vendor-level preset: one API key plus per-protocol
// endpoint variants and a model list. Projected via ProviderFromPreset it
// becomes a single vendor provider row that serves every agent adapter.
type ProviderPreset struct {
	Slug         string          `toml:"slug" json:"slug"`
	Name         string          `toml:"name" json:"name"`
	DefaultModel string          `toml:"default_model" json:"default_model"`
	Models       []string        `toml:"models" json:"models"`
	Variants     []PresetVariant `toml:"variants" json:"variants"`
}

// PresetVariant is one wire-protocol endpoint of a vendor preset.
type PresetVariant struct {
	Protocol     string                 `toml:"protocol" json:"protocol"`
	BaseURL      string                 `toml:"base_url" json:"base_url"`
	Endpoints    map[string]string      `toml:"endpoints" json:"endpoints"`
	Capabilities map[string]interface{} `toml:"capabilities" json:"capabilities"`
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

// ProviderFromPreset projects a vendor preset into a single provider row.
// The returned provider carries one API key, the preset's model list, and a
// variant per wire protocol — every agent resolves its own endpoint from it.
func ProviderFromPreset(preset ProviderPreset, apiKey string) store.Provider {
	defaultModel := preset.DefaultModel
	if defaultModel == "" && len(preset.Models) > 0 {
		defaultModel = preset.Models[0]
	}
	models := preset.Models
	if len(models) == 0 && defaultModel != "" {
		models = []string{defaultModel}
	}
	variants := make(map[string]store.ProviderVariant, len(preset.Variants))
	for _, variant := range preset.Variants {
		if variant.Protocol == "" || variant.BaseURL == "" {
			continue
		}
		variants[variant.Protocol] = store.ProviderVariant{
			BaseURL:      variant.BaseURL,
			Endpoints:    variant.Endpoints,
			Capabilities: variant.Capabilities,
		}
	}
	return store.Provider{
		Slug:         preset.Slug,
		Name:         preset.Name,
		APIKey:       apiKey,
		DefaultModel: defaultModel,
		Models:       models,
		Variants:     variants,
		Active:       true,
	}
}

// PresetLabel renders a one-line summary: vendor name, served protocols, and
// available models.
func PresetLabel(preset ProviderPreset) string {
	parts := []string{preset.Name + " ["}
	parts = append(parts, strings.Join(PresetProtocols(preset), ", "))
	parts = append(parts, "]")
	if len(preset.Models) > 0 {
		parts = append(parts, " models: "+strings.Join(preset.Models, ", "))
	}
	return strings.Join(parts, "")
}

// PresetProtocols lists the wire protocols a preset serves, in the canonical
// anthropic / openai_responses / openai_chat order when present.
func PresetProtocols(preset ProviderPreset) []string {
	seen := map[string]bool{}
	for _, variant := range preset.Variants {
		if variant.Protocol != "" {
			seen[variant.Protocol] = true
		}
	}
	protocols := make([]string, 0, len(seen))
	for _, protocol := range []string{"anthropic", "openai_responses", "openai_chat"} {
		if seen[protocol] {
			protocols = append(protocols, protocol)
		}
	}
	for protocol := range seen {
		if !contains(protocols, protocol) {
			protocols = append(protocols, protocol)
		}
	}
	return protocols
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func bytesReader(value []byte) *bytes.Reader {
	return bytes.NewReader(value)
}
