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
	if len(preset.URLOptions) != 3 {
		t.Fatalf("expected minimax to have 3 URL options, got %d", len(preset.URLOptions))
	}

	protocols := map[string]string{}
	models := map[string]string{}
	endpoints := map[string]map[string]string{}
	for _, option := range preset.URLOptions {
		protocols[option.Slug] = option.APIProtocol
		models[option.Slug] = option.DefaultModel
		endpoints[option.Slug] = option.Endpoints
	}
	if protocols["openai"] != "openai_chat" || protocols["claude"] != "anthropic" || protocols["codex"] != "openai_responses" {
		t.Fatalf("unexpected protocols: %+v", protocols)
	}
	if models["openai"] == "" || models["claude"] == "" || models["codex"] == "" {
		t.Fatalf("preset options must define default models: %+v", models)
	}
	if endpoints["openai"]["chat_completions"] == "" || endpoints["claude"]["messages"] == "" || endpoints["codex"]["responses"] == "" {
		t.Fatalf("preset options must define protocol endpoints: %+v", endpoints)
	}
}

func TestProviderPresetsVolcengineURLOptions(t *testing.T) {
	preset, err := FindPreset("volcengine")
	if err != nil {
		t.Fatal(err)
	}
	if len(preset.URLOptions) != 2 {
		t.Fatalf("expected volcengine to have 2 URL options, got %d", len(preset.URLOptions))
	}
	options := map[string]URLOption{}
	for _, option := range preset.URLOptions {
		options[option.Slug] = option
	}

	openai := options["openai"]
	if openai.APIProtocol != "openai_chat" {
		t.Fatalf("openai option protocol: %s", openai.APIProtocol)
	}
	if openai.BaseURL != "https://ark.cn-beijing.volces.com/api/v3" {
		t.Fatalf("openai option base url: %s", openai.BaseURL)
	}
	if openai.DefaultModel != "glm-5.2" {
		t.Fatalf("openai option default model: %s", openai.DefaultModel)
	}
	if openai.Endpoints["chat_completions"] != "/chat/completions" || openai.Endpoints["models"] != "/models" {
		t.Fatalf("openai option endpoints: %+v", openai.Endpoints)
	}

	claude := options["claude"]
	if claude.APIProtocol != "anthropic" {
		t.Fatalf("claude option protocol: %s", claude.APIProtocol)
	}
	if claude.BaseURL != "https://ark.cn-beijing.volces.com/api/plan" {
		t.Fatalf("claude option base url: %s", claude.BaseURL)
	}
	if claude.DefaultModel != "glm-5.2" {
		t.Fatalf("claude option default model: %s", claude.DefaultModel)
	}
	if claude.Endpoints["messages"] != "/v1/messages" || claude.Endpoints["models"] != "/v1/models" {
		t.Fatalf("claude option endpoints: %+v", claude.Endpoints)
	}
	extra, ok := claude.Capabilities["claude_extra_env"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected claude_extra_env capability, got: %+v", claude.Capabilities)
	}
	if extra["API_TIMEOUT_MS"] != "3000000" || extra["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] != "1" {
		t.Fatalf("unexpected claude_extra_env: %+v", extra)
	}

	// Each option slug differs from the preset slug, so ProviderFromPreset should
	// project them as "volcengine-openai" / "volcengine-claude" (compare minimax-*).
	openaiProvider := ProviderFromPreset(*preset, openai, "ark-test")
	if openaiProvider.Slug != "volcengine-openai" || openaiProvider.APIProtocol != "openai_chat" {
		t.Fatalf("unexpected openai provider: %s / %s", openaiProvider.Slug, openaiProvider.APIProtocol)
	}
	if openaiProvider.BaseURL != "https://ark.cn-beijing.volces.com/api/v3" {
		t.Fatalf("unexpected openai provider base url: %s", openaiProvider.BaseURL)
	}
	claudeProvider := ProviderFromPreset(*preset, claude, "ark-test")
	if claudeProvider.Slug != "volcengine-claude" || claudeProvider.APIProtocol != "anthropic" {
		t.Fatalf("unexpected claude provider: %s / %s", claudeProvider.Slug, claudeProvider.APIProtocol)
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
