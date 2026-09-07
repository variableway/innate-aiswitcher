package agentconfig

import (
	"context"
	"os/exec"
	"time"
)

// InstalledAgent reports whether an agent binary exists on this machine,
// where, and which version it reports.
type InstalledAgent struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Binary    string `json:"binary"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
}

// AgentBinaries maps agent slugs to the executable to look up.
var AgentBinaries = map[string]string{
	"claude":   "claude",
	"codex":    "codex",
	"opencode": "opencode",
}

// DiscoverAgents probes the local machine for each supported agent binary
// and collects its --version output (best effort, short timeout).
func DiscoverAgents() []InstalledAgent {
	result := make([]InstalledAgent, 0, len(AgentOrder))
	for _, slug := range AgentOrder {
		binary := AgentBinaries[slug]
		info := InstalledAgent{Slug: slug, Name: agentNames[slug], Binary: binary}
		if path, err := exec.LookPath(binary); err == nil {
			info.Installed = true
			info.Path = path
			info.Version = probeVersion(binary)
		}
		result = append(result, info)
	}
	return result
}

func probeVersion(binary string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// --version output goes to stdout or stderr depending on the tool
	out, err := exec.CommandContext(ctx, binary, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	version := trimAtMost(string(out), 120)
	if version == "" {
		return ""
	}
	return version
}

func trimAtMost(s string, max int) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\r' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\r' || s[end-1] == '\t') {
		end--
	}
	if end-start > max {
		end = start + max
	}
	return s[start:end]
}
