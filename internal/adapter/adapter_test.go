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
