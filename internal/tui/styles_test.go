package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/variableway/innate-aiswitcher/internal/adapter"
	"github.com/variableway/innate-aiswitcher/internal/store"
	"github.com/variableway/innate-aiswitcher/internal/templates"
)

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

// strip removes ANSI escape sequences so assertions match the visible text.
func strip(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

func sampleProvider() store.Provider {
	return store.Provider{
		Slug:         "minimax",
		Name:         "MiniMax",
		APIKey:       "sk-1234567890abcdef",
		APIProtocol:  "openai_chat",
		DefaultModel: "MiniMax-M3",
		Models:       []string{"MiniMax-M3", "MiniMax-M2"},
		Variants: map[string]store.ProviderVariant{
			"anthropic":        {BaseURL: "https://api.minimax.io/anthropic"},
			"openai_responses": {BaseURL: "https://api.minimax.io/v1"},
			"openai_chat":      {BaseURL: "https://api.minimax.io/v1"},
		},
		Active: true,
	}
}

func TestMaskAPIKey(t *testing.T) {
	if got := MaskAPIKey(""); got != "" {
		t.Fatalf("empty key should stay empty, got %q", got)
	}
	if got := MaskAPIKey("short"); got != "*****" {
		t.Fatalf("short key should be fully masked, got %q", got)
	}
	got := MaskAPIKey("sk-1234567890abcdef")
	if !strings.HasPrefix(got, "sk-1") || !strings.HasSuffix(got, "cdef") || strings.Contains(got, "567890") {
		t.Fatalf("long key should keep 4+4 chars and mask the middle, got %q", got)
	}
}

func TestKeyState(t *testing.T) {
	if got := strip(KeyState("sk-x")); got != "✓ key set" {
		t.Fatalf("set key: got %q", got)
	}
	if got := strip(KeyState("")); got != "✗ key missing" {
		t.Fatalf("missing key: got %q", got)
	}
}

func TestVariantProtocols(t *testing.T) {
	got := VariantProtocols(sampleProvider())
	want := "anthropic,openai_responses,openai_chat"
	if strings.Join(got, ",") != want {
		t.Fatalf("canonical protocol order: got %v want %v", got, want)
	}
	fallback := store.Provider{APIProtocol: "openai_chat"}
	if got := VariantProtocols(fallback); len(got) != 1 || got[0] != "openai_chat" {
		t.Fatalf("fallback to api_protocol: got %v", got)
	}
}

func TestProviderCard(t *testing.T) {
	out := strip(ProviderCard(sampleProvider()))
	for _, want := range []string{"minimax", "MiniMax", "anthropic", "MiniMax-M3", "✓ key set", "default model"} {
		if !strings.Contains(out, want) {
			t.Fatalf("provider card missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "key=") {
		t.Fatalf("card must not use key= wording:\n%s", out)
	}
	if strings.Contains(out, "sk-1234567890abcdef") {
		t.Fatalf("card must never print the raw API key:\n%s", out)
	}

	missing := sampleProvider()
	missing.APIKey = ""
	if out := strip(ProviderCard(missing)); !strings.Contains(out, "✗ key missing") {
		t.Fatalf("missing key state:\n%s", out)
	}
}

func TestRenderProviders(t *testing.T) {
	out := strip(RenderProviders([]store.Provider{sampleProvider()}))
	if !strings.Contains(out, "Saved Providers") || !strings.Contains(out, "Total: 1 provider(s)") {
		t.Fatalf("section title/hint missing:\n%s", out)
	}
	if out := strip(RenderProviders(nil)); !strings.Contains(out, "No providers yet") {
		t.Fatalf("empty state hint missing:\n%s", out)
	}
}

func TestRenderSavedProvider(t *testing.T) {
	out := strip(RenderSavedProvider(sampleProvider()))
	for _, want := range []string{"✓ Saved provider", "slug", "protocols", "anthropic"} {
		if !strings.Contains(out, want) {
			t.Fatalf("saved card missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "endpoints") {
		t.Fatalf("saved card must say protocols, not endpoints:\n%s", out)
	}
}

func TestRenderProfiles(t *testing.T) {
	profiles := []store.Profile{
		{Slug: "work", AgentSlug: "claude", ProviderSlug: "minimax", Model: "MiniMax-M3", IsDefault: true},
		{Slug: "backup", AgentSlug: "codex", ProviderSlug: "deepseek"},
	}
	out := strip(RenderProfiles(profiles))
	if !strings.Contains(out, "★ work") {
		t.Fatalf("default profile must carry ★:\n%s", out)
	}
	if strings.Contains(out, "★ backup") {
		t.Fatalf("non-default profile must not carry ★:\n%s", out)
	}
	if !strings.Contains(out, "(provider default)") {
		t.Fatalf("empty model should render as provider default:\n%s", out)
	}
	if out := strip(RenderProfiles(nil)); !strings.Contains(out, "No profiles yet") {
		t.Fatalf("empty state hint missing:\n%s", out)
	}
}

func TestRenderModels(t *testing.T) {
	out := strip(RenderModels(sampleProvider()))
	if !strings.Contains(out, "● MiniMax-M3 (default)") {
		t.Fatalf("default model must render with green ●:\n%s", out)
	}
	if !strings.Contains(out, "○ MiniMax-M2") {
		t.Fatalf("other models must render with ○:\n%s", out)
	}
}

func TestRenderPresetCard(t *testing.T) {
	preset := templates.ProviderPreset{
		Slug:   "minimax",
		Name:   "MiniMax",
		Models: []string{"MiniMax-M3"},
		Variants: []templates.PresetVariant{
			{Protocol: "anthropic", BaseURL: "https://api.minimax.io/anthropic"},
		},
	}
	out := strip(RenderPresetCard(preset, "builtin"))
	for _, want := range []string{"minimax", "MiniMax-M3", "anthropic", "builtin"} {
		if !strings.Contains(out, want) {
			t.Fatalf("preset card missing %q:\n%s", want, out)
		}
	}
}

func TestRenderSourcedPresets(t *testing.T) {
	presets := []templates.SourcedPreset{{
		ProviderPreset: templates.ProviderPreset{Slug: "custom", Name: "Custom"},
		Source:         "/home/user/.innate-aiswitcher/presets/custom.toml",
	}}
	out := strip(RenderSourcedPresets(presets))
	if !strings.Contains(out, "custom.toml") || !strings.Contains(out, "Total: 1 preset(s)") {
		t.Fatalf("sourced preset output:\n%s", out)
	}
}

func TestRenderEnvLines(t *testing.T) {
	env := map[string]string{
		"Z_VAR":    "plain",
		"API_KEY":  "sk-1234567890abcdef",
		"A_TOKEN":  "tok-1234567890",
		"BASE_URL": "https://api.example.test",
	}
	out := strip(RenderEnvLines(env))
	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 sorted lines, got %d:\n%s", len(lines), out)
	}
	// Sorted by key: API_KEY < A_TOKEN < BASE_URL < Z_VAR.
	if !strings.HasPrefix(lines[0], "API_KEY=") || !strings.HasPrefix(lines[1], "A_TOKEN=") || !strings.HasPrefix(lines[3], "Z_VAR=") {
		t.Fatalf("env lines must be key-sorted:\n%s", out)
	}
	if strings.Contains(out, "sk-1234567890abcdef") || strings.Contains(out, "tok-1234567890") {
		t.Fatalf("sensitive env values must be masked:\n%s", out)
	}
	if !strings.Contains(out, "BASE_URL=https://api.example.test") {
		t.Fatalf("non-sensitive env values stay readable:\n%s", out)
	}
	if RenderEnvLines(nil) != "" {
		t.Fatal("nil env renders empty")
	}
}

func TestRenderLaunchPlan(t *testing.T) {
	plan := adapter.LaunchPlan{
		Command: "claude",
		Args:    []string{"--dangerously-skip-permissions"},
		CWD:     "/tmp/project",
		Env: map[string]string{
			"ANTHROPIC_API_KEY":  "sk-1234567890abcdef",
			"ANTHROPIC_BASE_URL": "https://api.minimax.io/anthropic",
		},
		Files: map[string]string{"/tmp/aisw-claude.json": "{}"},
	}
	out := strip(RenderLaunchPlan("claude", "minimax", "", plan))
	for _, want := range []string{"Launch Plan", "agent", "claude", "provider", "minimax", "(provider default)", "command", "cwd", "env", "temp files"} {
		if !strings.Contains(out, want) {
			t.Fatalf("launch plan card missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "sk-1234567890abcdef") {
		t.Fatalf("launch plan must mask API keys in env:\n%s", out)
	}
	if !strings.Contains(out, "ANTHROPIC_BASE_URL=https://api.minimax.io/anthropic") {
		t.Fatalf("non-sensitive env should be readable:\n%s", out)
	}
}

func TestProviderSelectLabel(t *testing.T) {
	got := providerSelectLabel(sampleProvider())
	if got != "minimax · MiniMax" {
		t.Fatalf("select label should be 'slug · name', got %q", got)
	}
	if strings.Contains(got, "key=") {
		t.Fatalf("key= wording must be gone, got %q", got)
	}
}

func TestProviderDescription(t *testing.T) {
	got := providerDescription(sampleProvider())
	for _, want := range []string{"protocols: anthropic", "models: MiniMax-M3", "✓ key set"} {
		if !strings.Contains(got, want) {
			t.Fatalf("provider description missing %q: %q", want, got)
		}
	}
	empty := sampleProvider()
	empty.APIKey = ""
	if got := providerDescription(empty); !strings.Contains(got, "✗ key missing") {
		t.Fatalf("missing key description: %q", got)
	}
	if got := providerDescription(store.Provider{}); got != "" {
		t.Fatalf("zero provider renders empty description, got %q", got)
	}
}
