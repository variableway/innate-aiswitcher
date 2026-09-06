package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/variableway/innate-aiswitcher/internal/safefile"
	"github.com/variableway/innate-aiswitcher/internal/store"
)

// Preset sources: "builtin" presets ship inside the binary; "user" presets
// live as TOML files under the user presets directory and survive restarts.
const (
	PresetSourceBuiltin = "builtin"
	PresetSourceUser    = "user"
)

// UserPresetsDir is the directory holding user-saved preset files
// (~/.innate-aiswitcher/presets by default; AISW_PRESETS_DIR overrides).
func UserPresetsDir() (string, error) {
	if dir := os.Getenv("AISW_PRESETS_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".innate-aiswitcher", "presets"), nil
}

// SourcedPreset pairs a preset with its origin.
type SourcedPreset struct {
	ProviderPreset
	Source string `json:"source"`
}

// AllPresets returns builtin presets plus every user preset file, with user
// presets overriding builtin entries of the same slug (sorted by slug).
func AllPresets() ([]SourcedPreset, error) {
	bySlug := map[string]SourcedPreset{}
	builtin, err := ProviderPresets()
	if err != nil {
		return nil, err
	}
	for _, preset := range builtin {
		bySlug[preset.Slug] = SourcedPreset{ProviderPreset: preset, Source: PresetSourceBuiltin}
	}
	user, err := UserPresets()
	if err != nil {
		return nil, err
	}
	for _, preset := range user {
		bySlug[preset.Slug] = SourcedPreset{ProviderPreset: preset, Source: PresetSourceUser}
	}
	presets := make([]SourcedPreset, 0, len(bySlug))
	for _, preset := range bySlug {
		presets = append(presets, preset)
	}
	sort.Slice(presets, func(i, j int) bool { return presets[i].Slug < presets[j].Slug })
	return presets, nil
}

// UserPresets loads every *.toml file in the user presets directory.
func UserPresets() ([]ProviderPreset, error) {
	dir, err := UserPresetsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var presets []ProviderPreset
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		var data presetFile
		if _, err := toml.DecodeFile(filepath.Join(dir, entry.Name()), &data); err != nil {
			continue // skip malformed files, keep the rest loadable
		}
		presets = append(presets, data.Presets...)
	}
	return presets, nil
}

// FindSourcedPreset locates a preset by slug across builtin AND user files
// (user presets override builtin entries of the same slug).
func FindSourcedPreset(slug string) (*SourcedPreset, error) {
	presets, err := AllPresets()
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

// SaveUserPreset writes a preset to <dir>/<slug>.toml so it can be reloaded
// later from the file (and shared across machines).
func SaveUserPreset(preset ProviderPreset) (string, error) {
	if preset.Slug == "" {
		return "", fmt.Errorf("preset slug is required")
	}
	if len(preset.Variants) == 0 {
		return "", fmt.Errorf("preset %s needs at least one variant", preset.Slug)
	}
	dir, err := UserPresetsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, preset.Slug+".toml")
	var buf strings.Builder
	if err := toml.NewEncoder(&buf).Encode(presetFile{Presets: []ProviderPreset{preset}}); err != nil {
		return "", err
	}
	if err := safefile.Write(path, []byte(buf.String()), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// ImportUserPresetFile parses a preset TOML file and saves every preset it
// contains into the user presets directory.
func ImportUserPresetFile(path string) ([]ProviderPreset, error) {
	var data presetFile
	if _, err := toml.DecodeFile(path, &data); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if len(data.Presets) == 0 {
		return nil, fmt.Errorf("no [[presets]] found in %s", path)
	}
	for _, preset := range data.Presets {
		if _, err := SaveUserPreset(preset); err != nil {
			return nil, err
		}
	}
	return data.Presets, nil
}

// DeleteUserPreset removes a user preset file; builtin presets cannot be
// deleted.
func DeleteUserPreset(slug string) error {
	dir, err := UserPresetsDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, slug+".toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("user preset not found: %s", slug)
	}
	return os.Remove(path)
}

// PresetFromProvider derives a saveable preset from a stored provider row
// (API keys are intentionally NOT persisted in preset files).
func PresetFromProvider(provider store.Provider) ProviderPreset {
	preset := ProviderPreset{
		Slug:         provider.Slug,
		Name:         provider.Name,
		DefaultModel: provider.DefaultModel,
		Models:       provider.Models,
	}
	for protocol, variant := range provider.Variants {
		if protocol == "" || variant.BaseURL == "" {
			continue
		}
		preset.Variants = append(preset.Variants, PresetVariant{
			Protocol:     protocol,
			BaseURL:      variant.BaseURL,
			Endpoints:    variant.Endpoints,
			Capabilities: variant.Capabilities,
		})
	}
	if len(preset.Variants) == 0 && provider.BaseURL != "" {
		// Legacy single-protocol provider: keep it as a one-variant preset.
		preset.Variants = []PresetVariant{{
			Protocol:  provider.APIProtocol,
			BaseURL:   provider.BaseURL,
			Endpoints: provider.Endpoints,
		}}
	}
	sort.Slice(preset.Variants, func(i, j int) bool { return preset.Variants[i].Protocol < preset.Variants[j].Protocol })
	return preset
}
