package adapter

import (
	"fmt"
	"os"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

// PreviewFile is one generated configuration file for an agent session.
type PreviewFile struct {
	// Name is the agentconfig whitelist key (claude/settings, codex/config,
	// codex/auth) — empty when the projection is env-only.
	Name string `json:"name"`
	// Label is a display name, e.g. "settings.json".
	Label string `json:"label"`
	// Lang is a syntax hint for the UI.
	Lang string `json:"lang"`
	// Content is the exact text that aisw would write at launch.
	Content string `json:"content"`
	// Writable reports whether this file can be persisted via agentconfig.
	Writable bool `json:"writable"`
}

// PreviewResult is the full projection of a provider+model onto an agent:
// the temp files aisw would create and the env it would export.
type PreviewResult struct {
	Agent    string            `json:"agent"`
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Files    []PreviewFile     `json:"files"`
	Env      map[string]string `json:"env,omitempty"`
}

// previewLabels maps plan file keys to display metadata and the
// agentconfig whitelist name for optional persistence.
var previewLabels = map[string]struct {
	name     string
	label    string
	lang     string
	writable bool
}{
	"settings":   {"settings", "settings.json", "json", true},
	"config":     {"config", "config.toml", "toml", true},
	"auth":       {"auth", "auth.json", "json", true},
	"codex_home": {"", "CODEX_HOME (temp dir)", "", false},
}

// Preview projects agent+provider+model through the real launch pipeline
// (dry-run) and returns the exact configuration files and env vars the
// session would use — no process is started.
func Preview(agent store.Agent, provider store.Provider, model string) (PreviewResult, error) {
	plan, cleanup, err := BuildPlan(agent, provider, nil, LaunchOptions{DryRun: true, Model: model})
	if err != nil {
		return PreviewResult{}, err
	}
	defer cleanup()

	result := PreviewResult{
		Agent:    agent.Slug,
		Provider: provider.Slug,
		Model:    model,
		Env:      plan.Env,
	}
	for key, path := range plan.Files {
		meta, known := previewLabels[key]
		if !known {
			meta = struct {
				name     string
				label    string
				lang     string
				writable bool
			}{label: key, lang: ""}
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return PreviewResult{}, fmt.Errorf("stat preview file %s: %w", key, statErr)
		}
		content := ""
		if !info.IsDir() {
			bytes, err := os.ReadFile(path)
			if err != nil {
				return PreviewResult{}, fmt.Errorf("read preview file %s: %w", key, err)
			}
			content = string(bytes)
		}
		result.Files = append(result.Files, PreviewFile{
			Name:     meta.name,
			Label:    meta.label,
			Lang:     meta.lang,
			Content:  content,
			Writable: meta.writable,
		})
	}
	if len(result.Files) == 0 && len(result.Env) == 0 {
		return PreviewResult{}, fmt.Errorf("agent %s produced no previewable configuration", agent.Slug)
	}
	return result, nil
}
