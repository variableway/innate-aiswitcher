package adapter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

func TestBuildPlanProjectsSameProviderIntoAgentAdapters(t *testing.T) {
	provider := store.Provider{
		Slug:         "shared-openai",
		BaseURL:      "https://api.example.test/v1",
		APIKey:       "sk-shared",
		APIProtocol:  "openai_chat",
		DefaultModel: "gpt-shared",
	}
	profile := &store.Profile{
		Model:        "gpt-profile",
		DefaultArgs:  "--debug",
		EnvOverrides: map[string]string{"AISW_TEST": "1"},
	}

	claudePlan, claudeCleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, provider, profile, LaunchOptions{CWD: "/tmp/work"})
	if err != nil {
		t.Fatal(err)
	}
	defer claudeCleanup()

	codexPlan, codexCleanup, err := BuildPlan(store.Agent{Binary: "codex", Adapter: "codex"}, provider, profile, LaunchOptions{CWD: "/tmp/work"})
	if err != nil {
		t.Fatal(err)
	}
	defer codexCleanup()

	if !strings.Contains(claudePlan.Command, "claude --settings") || !strings.Contains(claudePlan.Command, "--debug") {
		t.Fatalf("unexpected claude command: %s", claudePlan.Command)
	}
	if !strings.Contains(codexPlan.Command, "codex --debug") {
		t.Fatalf("unexpected codex command: %s", codexPlan.Command)
	}
	if claudePlan.Env["CODEX_HOME"] != "" {
		t.Fatalf("claude plan should not set CODEX_HOME: %+v", claudePlan.Env)
	}
	if codexPlan.Env["CODEX_HOME"] == "" {
		t.Fatalf("codex plan should set CODEX_HOME: %+v", codexPlan.Env)
	}
}

func TestBuildClaudePlanWritesSessionSettings(t *testing.T) {
	provider := store.Provider{
		Slug:         "anthropic-compatible",
		BaseURL:      "https://anthropic.example.test",
		APIKey:       "anthropic-key",
		APIProtocol:  "anthropic",
		DefaultModel: "claude-default",
	}
	profile := &store.Profile{
		Model:           "claude-profile",
		ConfigOverrides: map[string]any{"theme": "quiet"},
		EnvOverrides:    map[string]string{"EXTRA_ENV": "enabled"},
	}

	plan, cleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, provider, profile, LaunchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	settingsPath := plan.Files["settings"]
	bytes, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(bytes, &settings); err != nil {
		t.Fatal(err)
	}
	env := settings["env"].(map[string]any)
	if env["ANTHROPIC_AUTH_TOKEN"] != "anthropic-key" || env["ANTHROPIC_MODEL"] != "claude-profile" {
		t.Fatalf("unexpected claude env: %+v", env)
	}
	if env["EXTRA_ENV"] != "enabled" || settings["theme"] != "quiet" {
		t.Fatalf("profile overrides were not merged: %+v", settings)
	}
}

func TestBuildCodexPlanWritesEphemeralHome(t *testing.T) {
	provider := store.Provider{
		Slug:         "Mini Max",
		BaseURL:      "https://api.minimax.example/v1",
		APIKey:       "sk-minimax",
		APIProtocol:  "openai_responses",
		DefaultModel: "minimax-default",
	}
	profile := &store.Profile{
		Model: "minimax-profile",
		ConfigOverrides: map[string]any{
			"codex_config_append": "approval_policy = \"never\"",
		},
	}

	plan, cleanup, err := BuildPlan(store.Agent{Binary: "codex", Adapter: "codex"}, provider, profile, LaunchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	codexHome := plan.Env["CODEX_HOME"]
	if codexHome == "" {
		t.Fatalf("expected CODEX_HOME env: %+v", plan.Env)
	}

	config, err := os.ReadFile(plan.Files["config"])
	if err != nil {
		t.Fatal(err)
	}
	configText := string(config)
	for _, want := range []string{`model_provider = "mini-max"`, `model = "minimax-profile"`, `wire_api = "responses"`, `approval_policy = "never"`} {
		if !strings.Contains(configText, want) {
			t.Fatalf("config missing %q:\n%s", want, configText)
		}
	}

	auth, err := os.ReadFile(plan.Files["auth"])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(auth), "sk-minimax") {
		t.Fatalf("auth file does not contain provider key: %s", auth)
	}

	cleanup()
	if _, err := os.Stat(codexHome); !os.IsNotExist(err) {
		t.Fatalf("expected cleanup to remove codex home, got err=%v", err)
	}
}

func TestBuildPlanRequiresExplicitModel(t *testing.T) {
	provider := store.Provider{
		Slug:        "missing-model",
		BaseURL:     "https://api.example.test/v1",
		APIKey:      "sk-test",
		APIProtocol: "openai_chat",
	}
	plan, cleanup, err := BuildPlan(store.Agent{Binary: "codex", Adapter: "codex"}, provider, nil, LaunchOptions{})
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatalf("expected missing model error, got plan %+v", plan)
	}
	if !strings.Contains(err.Error(), "default model") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildGeminiPlan(t *testing.T) {
	provider := store.Provider{
		Slug:         "gemini-provider",
		BaseURL:      "https://generativelanguage.googleapis.com",
		APIKey:       "gemini-key",
		APIProtocol:  "openai_chat",
		DefaultModel: "gemini-2.0",
	}
	plan, cleanup, err := BuildPlan(store.Agent{Binary: "gemini", Adapter: "gemini"}, provider, nil, LaunchOptions{CWD: "/tmp/work"})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if plan.Command != "gemini" {
		t.Fatalf("unexpected command: %s", plan.Command)
	}
	if plan.Env["GEMINI_API_KEY"] != "gemini-key" {
		t.Fatalf("unexpected GEMINI_API_KEY: %s", plan.Env["GEMINI_API_KEY"])
	}
	if plan.Env["GOOGLE_GEMINI_BASE_URL"] != "https://generativelanguage.googleapis.com" {
		t.Fatalf("unexpected base url: %s", plan.Env["GOOGLE_GEMINI_BASE_URL"])
	}
	if plan.CWD != "/tmp/work" {
		t.Fatalf("unexpected cwd: %s", plan.CWD)
	}
}

func TestBuildOpenAIEnvPlan(t *testing.T) {
	provider := store.Provider{
		Slug:         "openai-provider",
		BaseURL:      "https://api.openai.com/v1",
		APIKey:       "openai-key",
		APIProtocol:  "openai_chat",
		DefaultModel: "gpt-4",
	}
	profile := &store.Profile{
		EnvOverrides: map[string]string{"EXTRA": "value"},
	}
	plan, cleanup, err := BuildPlan(store.Agent{Binary: "trae", Adapter: "openai_env"}, provider, profile, LaunchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if plan.Env["OPENAI_API_KEY"] != "openai-key" {
		t.Fatalf("unexpected OPENAI_API_KEY: %s", plan.Env["OPENAI_API_KEY"])
	}
	if plan.Env["OPENAI_BASE_URL"] != "https://api.openai.com/v1" {
		t.Fatalf("unexpected OPENAI_BASE_URL: %s", plan.Env["OPENAI_BASE_URL"])
	}
	if plan.Env["EXTRA"] != "value" {
		t.Fatalf("profile env override not applied: %s", plan.Env["EXTRA"])
	}
}

func TestBuildPlanUnsupportedAdapter(t *testing.T) {
	provider := store.Provider{
		Slug:         "test",
		BaseURL:      "https://api.example.test",
		APIKey:       "sk-test",
		APIProtocol:  "openai_chat",
		DefaultModel: "gpt-test",
	}
	_, cleanup, err := BuildPlan(store.Agent{Binary: "unknown", Adapter: "unknown_adapter"}, provider, nil, LaunchOptions{})
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatal("expected error for unsupported adapter")
	}
	if !strings.Contains(err.Error(), "unsupported adapter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuilderNames(t *testing.T) {
	names := BuilderNames()
	if len(names) == 0 {
		t.Fatal("expected non-empty builder names")
	}
	seen := map[string]bool{}
	for _, name := range names {
		seen[name] = true
	}
	for _, required := range []string{"claude", "codex", "gemini", "openai_env"} {
		if !seen[required] {
			t.Fatalf("expected builder %q to be registered", required)
		}
	}
}

func TestRegister(t *testing.T) {
	Register("test_adapter", func(ctx BuildContext) (LaunchPlan, func(), error) {
		return LaunchPlan{Command: "test"}, func() {}, nil
	})

	provider := store.Provider{
		Slug:         "test",
		BaseURL:      "https://api.example.test",
		APIKey:       "sk-test",
		APIProtocol:  "openai_chat",
		DefaultModel: "gpt-test",
	}
	plan, cleanup, err := BuildPlan(store.Agent{Binary: "test", Adapter: "test_adapter"}, provider, nil, LaunchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if plan.Command != "test" {
		t.Fatalf("unexpected command: %s", plan.Command)
	}
}

func TestDefaultArgs(t *testing.T) {
	if got := defaultArgs(nil); got != nil {
		t.Fatalf("expected nil for nil profile, got %v", got)
	}
	if got := defaultArgs(&store.Profile{}); got != nil {
		t.Fatalf("expected nil for empty args, got %v", got)
	}
	got := defaultArgs(&store.Profile{DefaultArgs: "--foo --bar"})
	if len(got) != 2 || got[0] != "--foo" || got[1] != "--bar" {
		t.Fatalf("unexpected args: %v", got)
	}
}

func TestEnvOverrides(t *testing.T) {
	if got := envOverrides(nil); len(got) != 0 {
		t.Fatalf("expected empty map for nil profile, got %v", got)
	}
	if got := envOverrides(&store.Profile{}); len(got) != 0 {
		t.Fatalf("expected empty map for empty overrides, got %v", got)
	}
	got := envOverrides(&store.Profile{EnvOverrides: map[string]string{"K": "V"}})
	if got["K"] != "V" {
		t.Fatalf("unexpected overrides: %v", got)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "''"},
		{"hello", "hello"},
		{"hello world", "'hello world'"},
		{"it's", "'it'\\''s'"},
		{"a-b_c.d,e/f:@%+", "a-b_c.d,e/f:@%+"},
	}
	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestJoinCommand(t *testing.T) {
	got := joinCommand([]string{"claude", "--settings", "/tmp/a b", "--debug"})
	// Simple args like --settings are not quoted; args with spaces are quoted.
	if strings.Contains(got, "'--settings'") {
		t.Fatalf("--settings should not be quoted, got: %s", got)
	}
	if !strings.Contains(got, "'/tmp/a b'") {
		t.Fatalf("expected space-quoted path, got: %s", got)
	}
}

func TestSanitizeProviderKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Mini Max", "mini-max"},
		{"OpenAI", "openai"},
		{"---test---", "test"},
		{"a_b_c", "a-b-c"},
		{"", "provider"},
		{"!!!", "provider"},
		{"API-Key-123", "api-key-123"},
	}
	for _, tt := range tests {
		got := sanitizeProviderKey(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeProviderKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWithEnvPrefix(t *testing.T) {
	if got := withEnvPrefix("cmd", nil); got != "cmd" {
		t.Fatalf("expected cmd without env, got %s", got)
	}
	got := withEnvPrefix("cmd", map[string]string{"K1": "v1", "K2": "v 2"})
	if !strings.HasPrefix(got, "K1=v1 ") && !strings.HasPrefix(got, "K2='v 2' ") {
		t.Fatalf("unexpected env prefix: %s", got)
	}
	if !strings.HasSuffix(got, " cmd") {
		t.Fatalf("expected command suffix, got %s", got)
	}
}
