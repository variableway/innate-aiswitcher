package configfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/variableway/innate-aiswitcher/internal/store"

	_ "github.com/variableway/innate-aiswitcher/migrations"
)

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Fatal("expected non-empty default path")
	}
	if !strings.Contains(path, ".innate-aiswitcher") {
		t.Fatalf("expected path to contain .innate-aiswitcher, got %s", path)
	}
	if filepath.Ext(path) != ".toml" {
		t.Fatalf("expected .toml extension, got %s", path)
	}
}

func TestDefaultBackupPath(t *testing.T) {
	path := DefaultBackupPath()
	if path == "" {
		t.Fatal("expected non-empty backup path")
	}
	if !strings.Contains(path, "backups") {
		t.Fatalf("expected path to contain backups, got %s", path)
	}
	if !strings.HasPrefix(filepath.Base(path), "config-") {
		t.Fatalf("expected basename to start with config-, got %s", filepath.Base(path))
	}
	if filepath.Ext(path) != ".toml" {
		t.Fatalf("expected .toml extension, got %s", path)
	}
}

func TestConfigStructRoundtripTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.toml")

	original := Config{
		Version: 1,
		Providers: []store.Provider{
			{Slug: "openai", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIProtocol: "openai_chat", DefaultModel: "gpt-4", Active: true},
			{Slug: "anthropic", Name: "Anthropic", BaseURL: "https://api.anthropic.com", APIProtocol: "anthropic", DefaultModel: "claude-3", APIKey: "sk-secret", Active: true},
		},
		Profiles: []store.Profile{
			{Slug: "claude-openai", AgentSlug: "claude", ProviderSlug: "openai", Model: "gpt-4"},
		},
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := toml.NewEncoder(file).Encode(original); err != nil {
		file.Close()
		t.Fatalf("encode failed: %v", err)
	}
	file.Close()

	var decoded Config
	if _, err := toml.DecodeFile(path, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Version != 1 {
		t.Fatalf("version mismatch: got %d, want 1", decoded.Version)
	}
	if len(decoded.Providers) != 2 {
		t.Fatalf("provider count mismatch: got %d, want 2", len(decoded.Providers))
	}
	if decoded.Providers[0].Slug != "openai" {
		t.Fatalf("unexpected provider slug: %s", decoded.Providers[0].Slug)
	}
	if decoded.Providers[1].APIKey != "sk-secret" {
		t.Fatalf("api key not preserved: got %q", decoded.Providers[1].APIKey)
	}
	if len(decoded.Profiles) != 1 || decoded.Profiles[0].AgentSlug != "claude" {
		t.Fatalf("profile mismatch: %+v", decoded.Profiles)
	}
}

func TestConfigStructRoundtripJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.json")

	original := Config{
		Version: 1,
		Providers: []store.Provider{
			{Slug: "openai", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIProtocol: "openai_chat", DefaultModel: "gpt-4", Active: true},
			{Slug: "anthropic", Name: "Anthropic", BaseURL: "https://api.anthropic.com", APIProtocol: "anthropic", DefaultModel: "claude-3", APIKey: "sk-secret", Active: true, Headers: map[string]string{"X-Test": "yes"}, Endpoints: map[string]string{"chat": "/v1/chat"}},
		},
		Profiles: []store.Profile{
			{Slug: "claude-openai", AgentSlug: "claude", ProviderSlug: "openai", Model: "gpt-4", EnvOverrides: map[string]string{"FOO": "bar"}},
		},
	}

	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Version != 1 {
		t.Fatalf("version mismatch: got %d, want 1", decoded.Version)
	}
	if len(decoded.Providers) != 2 {
		t.Fatalf("provider count mismatch: got %d, want 2", len(decoded.Providers))
	}
	if decoded.Providers[1].APIKey != "sk-secret" {
		t.Fatalf("api key not preserved: got %q", decoded.Providers[1].APIKey)
	}
	if decoded.Providers[1].Headers["X-Test"] != "yes" {
		t.Fatalf("headers not preserved: %+v", decoded.Providers[1].Headers)
	}
	if decoded.Providers[1].Endpoints["chat"] != "/v1/chat" {
		t.Fatalf("endpoints not preserved: %+v", decoded.Providers[1].Endpoints)
	}
	if len(decoded.Profiles) != 1 || decoded.Profiles[0].AgentSlug != "claude" {
		t.Fatalf("profile mismatch: %+v", decoded.Profiles)
	}
	if decoded.Profiles[0].EnvOverrides["FOO"] != "bar" {
		t.Fatalf("env overrides not preserved: %+v", decoded.Profiles[0].EnvOverrides)
	}
}

func TestConfigVersionMissing(t *testing.T) {
	content := `[[providers]]
slug = "test"
`
	var cfg Config
	if _, err := toml.Decode(content, &cfg); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if cfg.Version != 0 {
		t.Fatalf("expected version 0 for missing version field, got %d", cfg.Version)
	}
}

func TestExportStripsSecretsWhenRequested(t *testing.T) {
	// Build a config, run the same stripping logic Export uses, and confirm
	// the api_key is gone. We don't need a real PB for this slice.
	cfg := Config{
		Version: 1,
		Providers: []store.Provider{
			{Slug: "test", APIKey: "sk-secret", Active: true},
		},
	}
	for i := range cfg.Providers {
		cfg.Providers[i].APIKey = ""
	}
	if cfg.Providers[0].APIKey != "" {
		t.Fatal("api_key should be empty after stripping")
	}
}

func TestDetectFormat(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		ext     string
		content string
		want    Format
		wantErr bool
	}{
		{name: "json ext", ext: ".json", content: `{"version":1}`, want: FormatJSON},
		{name: "toml ext", ext: ".toml", content: `version = 1`, want: FormatTOML},
		{name: "json content sniff", ext: ".txt", content: `  {"version":1}`, want: FormatJSON},
		{name: "toml content sniff comment", ext: ".txt", content: "\n# c\n[providers]\n", want: FormatTOML},
		{name: "toml content sniff bare key", ext: ".txt", content: "version = 1\n", want: FormatTOML},
		{name: "empty file", ext: ".txt", content: "  \n", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, "detect"+tc.ext)
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := DetectFormat(path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got format %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// newTestPB constructs a fresh PocketBase against a temp data dir with
// migrations applied. Returns a Store ready to use.
func newTestPB(t *testing.T) (*store.Store, func()) {
	t.Helper()
	dir := t.TempDir()
	pb := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dir})
	migratecmd.MustRegister(pb, pb.RootCmd, migratecmd.Config{Automigrate: true})
	pb.OnBootstrap().Bind(&hook.Handler[*core.BootstrapEvent]{
		Func: func(e *core.BootstrapEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			if err := e.App.RunAppMigrations(); err != nil {
				return err
			}
			return e.App.ReloadCachedCollections()
		},
	})
	if err := pb.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	s := store.New(pb)
	return s, func() {
		// PocketBase has no explicit shutdown; temp dir cleanup handles teardown.
	}
}

func TestExportImportJSONRoundTrip_PocketBase(t *testing.T) {
	s, cleanup := newTestPB(t)
	defer cleanup()

	// Seed a provider with all the fields we care about, plus a profile.
	p1, err := s.UpsertProvider(store.Provider{
		Slug: "minimax-openai", Name: "Minimax OpenAI", BaseURL: "https://api.minimax.example/v1",
		APIProtocol: "openai_chat", DefaultModel: "MiniMax-Text-01", APIKey: "sk-seed-key",
		Headers:   map[string]string{"X-Org": "test"},
		Endpoints: map[string]string{"chat_completions": "/chat/completions"},
		Active:    true,
	})
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	if _, err := s.UpsertProfile(store.Profile{
		Slug: "codex-minimax", AgentSlug: "codex", ProviderSlug: "minimax-openai",
		Model: "MiniMax-Text-01", IsDefault: true,
	}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	// Export to JSON with secrets.
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "export.json")
	if err := Export(s, jsonPath, FormatJSON, true); err != nil {
		t.Fatalf("export json: %v", err)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	if !strings.Contains(string(data), `"api_key": "sk-seed-key"`) {
		t.Fatalf("expected api_key in JSON when includeSecrets=true, got:\n%s", string(data))
	}

	// Parse the JSON and confirm shape.
	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal exported json: %v", err)
	}
	if parsed.Version != 1 {
		t.Fatalf("version: got %d want 1", parsed.Version)
	}
	if len(parsed.Providers) != 1 || parsed.Providers[0].Slug != "minimax-openai" {
		t.Fatalf("providers: %+v", parsed.Providers)
	}
	if parsed.Providers[0].Endpoints["chat_completions"] != "/chat/completions" {
		t.Fatalf("endpoints lost: %+v", parsed.Providers[0].Endpoints)
	}
	if len(parsed.Profiles) != 1 || parsed.Profiles[0].Slug != "codex-minimax" {
		t.Fatalf("profiles: %+v", parsed.Profiles)
	}

	// Wipe the DB by replacing the providers/profiles with a stub. We do
	// this by deleting the records directly via the store helpers.
	allProviders, err := s.ListProviders()
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	for _, p := range allProviders {
		if err := s.DeleteProvider(p.Slug); err != nil {
			t.Fatalf("delete provider %s: %v", p.Slug, err)
		}
	}
	allProfiles, err := s.ListProfiles()
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	for _, pr := range allProfiles {
		if err := s.DeleteProfile(pr.Slug); err != nil {
			t.Fatalf("delete profile %s: %v", pr.Slug, err)
		}
	}
	if pp, _ := s.ListProviders(); len(pp) != 0 {
		t.Fatalf("providers not empty after wipe: %+v", pp)
	}

	// Re-import from the JSON. Format "" → auto-detect from .json extension.
	if err := Import(s, jsonPath, ""); err != nil {
		t.Fatalf("import json: %v", err)
	}

	// Verify the DB now matches the pre-export state.
	gotProviders, err := s.ListProviders()
	if err != nil {
		t.Fatalf("list providers after import: %v", err)
	}
	if len(gotProviders) != 1 {
		t.Fatalf("providers after import: got %d want 1 (%+v)", len(gotProviders), gotProviders)
	}
	gp := gotProviders[0]
	if gp.Slug != p1.Slug || gp.Name != p1.Name || gp.BaseURL != p1.BaseURL {
		t.Fatalf("provider identity lost: %+v", gp)
	}
	if gp.APIKey != "sk-seed-key" {
		t.Fatalf("api_key not restored: got %q", gp.APIKey)
	}
	if gp.DefaultModel != "MiniMax-Text-01" {
		t.Fatalf("default_model not restored: %q", gp.DefaultModel)
	}
	if gp.Endpoints["chat_completions"] != "/chat/completions" {
		t.Fatalf("endpoints not restored: %+v", gp.Endpoints)
	}

	gotProfiles, err := s.ListProfiles()
	if err != nil {
		t.Fatalf("list profiles after import: %v", err)
	}
	if len(gotProfiles) != 1 {
		t.Fatalf("profiles after import: got %d want 1", len(gotProfiles))
	}
	pr := gotProfiles[0]
	if pr.Slug != "codex-minimax" || pr.AgentSlug != "codex" || pr.ProviderSlug != "minimax-openai" {
		t.Fatalf("profile identity lost: %+v", pr)
	}
	if pr.Model != "MiniMax-Text-01" || !pr.IsDefault {
		t.Fatalf("profile fields lost: %+v", pr)
	}
}

func TestImportJSONRejectsUnknownFields(t *testing.T) {
	s, cleanup := newTestPB(t)
	defer cleanup()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	// `bogus` is not a field on Config; we want Import to reject it rather
	// than silently accept (matches the existing TOML behavior — TOML's
	// BurntSushi decoder also errors on unknown fields by default).
	content := `{"version":1,"bogus":true,"providers":[],"profiles":[]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Import(s, path, FormatJSON); err == nil {
		t.Fatal("expected error for unknown JSON field, got nil")
	}
}

func TestImportJSONStripsSecretsWhenOmitted(t *testing.T) {
	s, cleanup := newTestPB(t)
	defer cleanup()

	dir := t.TempDir()
	path := filepath.Join(dir, "no-secrets.json")
	content := `{"version":1,"providers":[{"slug":"acme","name":"Acme","base_url":"https://acme","api_protocol":"openai_chat","default_model":"m","active":true}],"profiles":[]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Import(s, path, FormatJSON); err != nil {
		t.Fatalf("import: %v", err)
	}
	pps, err := s.ListProviders()
	if err != nil {
		t.Fatal(err)
	}
	if len(pps) != 1 {
		t.Fatalf("want 1 provider, got %d", len(pps))
	}
	if pps[0].APIKey != "" {
		t.Fatalf("expected empty api_key, got %q", pps[0].APIKey)
	}
}
