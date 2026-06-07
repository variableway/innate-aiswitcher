package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderPresetsIncludeMinimaxURLChoices(t *testing.T) {
	preset, err := FindPreset("minimax")
	if err != nil {
		t.Fatal(err)
	}
	if len(preset.URLOptions) != 2 {
		t.Fatalf("expected minimax to have 2 URL options, got %d", len(preset.URLOptions))
	}

	protocols := map[string]string{}
	models := map[string]string{}
	endpoints := map[string]map[string]string{}
	for _, option := range preset.URLOptions {
		protocols[option.Slug] = option.APIProtocol
		models[option.Slug] = option.DefaultModel
		endpoints[option.Slug] = option.Endpoints
	}
	if protocols["openai"] != "openai_chat" || protocols["claude"] != "anthropic" {
		t.Fatalf("unexpected protocols: %+v", protocols)
	}
	if models["openai"] == "" || models["claude"] == "" {
		t.Fatalf("preset options must define default models: %+v", models)
	}
	if endpoints["openai"]["chat_completions"] == "" || endpoints["claude"]["messages"] == "" {
		t.Fatalf("preset options must define protocol endpoints: %+v", endpoints)
	}
}

func TestConfigExampleIsEmbedded(t *testing.T) {
	content, err := ConfigExample()
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("expected embedded config template")
	}
}

func TestFindPresetNotFound(t *testing.T) {
	_, err := FindPreset("nonexistent-provider")
	if err == nil {
		t.Fatal("expected error for nonexistent preset")
	}
}

func TestProviderFromPreset(t *testing.T) {
	preset := ProviderPreset{Slug: "minimax", Name: "MiniMax"}
	option := URLOption{Slug: "openai", Label: "OpenAI", BaseURL: "https://api.minimax.ai/v1", APIProtocol: "openai_chat", DefaultModel: "abab6.5s"}
	provider := ProviderFromPreset(preset, option, "sk-test")

	if provider.Slug != "minimax-openai" {
		t.Fatalf("unexpected slug: %s", provider.Slug)
	}
	if provider.Name != "MiniMax OpenAI" {
		t.Fatalf("unexpected name: %s", provider.Name)
	}
	if provider.BaseURL != option.BaseURL {
		t.Fatalf("unexpected base url: %s", provider.BaseURL)
	}
	if provider.APIKey != "sk-test" {
		t.Fatalf("unexpected api key: %s", provider.APIKey)
	}
	if provider.APIProtocol != "openai_chat" {
		t.Fatalf("unexpected protocol: %s", provider.APIProtocol)
	}
	if provider.DefaultModel != "abab6.5s" {
		t.Fatalf("unexpected model: %s", provider.DefaultModel)
	}
	if !provider.Active {
		t.Fatal("expected provider to be active")
	}
}

func TestProviderFromPresetSingleOption(t *testing.T) {
	preset := ProviderPreset{Slug: "openai", Name: "OpenAI"}
	option := URLOption{Slug: "openai", Label: "Standard", BaseURL: "https://api.openai.com/v1", APIProtocol: "openai_chat", DefaultModel: "gpt-4"}
	provider := ProviderFromPreset(preset, option, "sk-key")

	// When option slug matches preset slug and there's only one option, keep original slug.
	if provider.Slug != "openai" {
		t.Fatalf("unexpected slug for single option: %s", provider.Slug)
	}
	if provider.Name != "OpenAI" {
		t.Fatalf("unexpected name for single option: %s", provider.Name)
	}
}

func TestPresetLabel(t *testing.T) {
	preset := ProviderPreset{Slug: "test", Name: "Test", URLOptions: []URLOption{
		{Label: "OpenAI"},
		{Label: "Claude"},
	}}
	label := PresetLabel(preset)
	if label != "Test (OpenAI, Claude)" {
		t.Fatalf("unexpected label: %s", label)
	}

	empty := ProviderPreset{Slug: "empty", Name: "Empty"}
	if PresetLabel(empty) != "Empty (no URL options)" {
		t.Fatalf("unexpected empty label: %s", PresetLabel(empty))
	}
}

func TestOptionLabel(t *testing.T) {
	option := URLOption{Label: "Standard", APIProtocol: "openai_chat", BaseURL: "https://api.openai.com/v1"}
	label := OptionLabel(option)
	if label != "Standard - openai_chat - https://api.openai.com/v1" {
		t.Fatalf("unexpected label: %s", label)
	}
}

func TestWriteConfigExample(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteConfigExample(path); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty config example")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected permissions: got %o, want %o", info.Mode().Perm(), 0o600)
	}
}

func TestProviderPresetsSorted(t *testing.T) {
	presets, err := ProviderPresets()
	if err != nil {
		t.Fatal(err)
	}
	if len(presets) == 0 {
		t.Fatal("expected at least one preset")
	}
	for i := 1; i < len(presets); i++ {
		if presets[i].Slug < presets[i-1].Slug {
			t.Fatalf("presets not sorted: %s before %s", presets[i-1].Slug, presets[i].Slug)
		}
	}
}
