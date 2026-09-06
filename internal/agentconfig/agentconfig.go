// Package agentconfig exposes the real on-disk configuration files of the
// supported coding agents (claude code, codex, opencode) so the Web UI can
// view, edit and save them directly. Only whitelisted files are reachable;
// saves go through safefile for atomic, 0o600 writes.
package agentconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/variableway/innate-aiswitcher/internal/safefile"
)

// FileSpec describes one editable agent configuration file.
type FileSpec struct {
	Agent string `json:"agent"`
	Name  string `json:"name"`  // short key used in URLs, e.g. "settings"
	Path  string `json:"path"`  // absolute on-disk path
	Lang  string `json:"lang"`  // syntax hint: json | toml
	Label string `json:"label"` // display name
}

// AgentInfo groups the editable files of one agent.
type AgentInfo struct {
	Agent string     `json:"agent"`
	Name  string     `json:"name"`
	Files []FileInfo `json:"files"`
}

// FileInfo is a FileSpec plus its current content.
type FileInfo struct {
	FileSpec
	Exists   bool   `json:"exists"`
	Content  string `json:"content"`
	Editable bool   `json:"editable"`
}

// Specs is the whitelist of editable configuration files.
var Specs = []FileSpec{
	{Agent: "claude", Name: "settings", Label: "settings.json", Lang: "json"},
	{Agent: "codex", Name: "config", Label: "config.toml", Lang: "toml"},
	{Agent: "codex", Name: "auth", Label: "auth.json", Lang: "json"},
	{Agent: "opencode", Name: "opencode", Label: "opencode.json", Lang: "json"},
}

// agentNames maps the agent slug to its display name.
var agentNames = map[string]string{
	"claude":   "Claude Code",
	"codex":    "Codex CLI",
	"opencode": "OpenCode",
}

// AgentOrder is the display order of agents.
var AgentOrder = []string{"claude", "codex", "opencode"}

// pathFor resolves the on-disk path of a whitelisted file. The "claude"
// entry is a synthetic marker; the real file is "settings".
func pathFor(spec FileSpec) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot determine home directory")
	}
	switch {
	case spec.Agent == "claude" && spec.Name == "settings":
		return filepath.Join(home, ".claude", "settings.json"), nil
	case spec.Agent == "codex" && spec.Name == "config":
		return filepath.Join(home, ".codex", "config.toml"), nil
	case spec.Agent == "codex" && spec.Name == "auth":
		return filepath.Join(home, ".codex", "auth.json"), nil
	case spec.Agent == "opencode" && spec.Name == "opencode":
		return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
	}
	return "", fmt.Errorf("unknown agent config: %s/%s", spec.Agent, spec.Name)
}

// List returns every editable file grouped by agent with its content.
func List() ([]AgentInfo, error) {
	result := make([]AgentInfo, 0, len(AgentOrder))
	for _, agent := range AgentOrder {
		info := AgentInfo{Agent: agent, Name: agentNames[agent]}
		for _, spec := range Specs {
			if spec.Agent != agent {
				continue
			}
			path, err := pathFor(spec)
			if err != nil {
				return nil, err
			}
			content, exists := readFile(path)
			spec.Path = path
			info.Files = append(info.Files, FileInfo{
				FileSpec: spec,
				Exists:   exists,
				Content:  content,
				Editable: true,
			})
		}
		result = append(result, info)
	}
	return result, nil
}

// Read returns the content of one whitelisted file.
func Read(agent, name string) (FileInfo, error) {
	spec, path, err := lookup(agent, name)
	if err != nil {
		return FileInfo{}, err
	}
	content, exists := readFile(path)
	spec.Path = path
	return FileInfo{FileSpec: spec, Exists: exists, Content: content, Editable: true}, nil
}

// Write atomically saves new content to a whitelisted file, creating parent
// directories as needed.
func Write(agent, name, content string) (FileInfo, error) {
	spec, path, err := lookup(agent, name)
	if err != nil {
		return FileInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return FileInfo{}, err
	}
	if err := safefile.Write(path, []byte(content), 0o600); err != nil {
		return FileInfo{}, err
	}
	spec.Path = path
	return FileInfo{FileSpec: spec, Exists: true, Content: content, Editable: true}, nil
}

func lookup(agent, name string) (FileSpec, string, error) {
	for _, spec := range Specs {
		if spec.Agent == agent && spec.Name == name {
			path, err := pathFor(spec)
			if err != nil {
				return FileSpec{}, "", err
			}
			return spec, path, nil
		}
	}
	return FileSpec{}, "", fmt.Errorf("unknown agent config: %s/%s", agent, name)
}

func readFile(path string) (string, bool) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(bytes), true
}
