// Package adapter provides launch plan builders for AI agents.
// Each adapter translates a shared Provider into agent-specific temporary
// configurations, environment variables, or settings files.
package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/variableway/innate-aiswitcher/internal/safefile"
	"github.com/variableway/innate-aiswitcher/internal/store"
)

type LaunchOptions struct {
	CWD      string
	Terminal string
	DryRun   bool
	Args     []string
}

type LaunchPlan struct {
	Command string            `json:"command"`
	CWD     string            `json:"cwd"`
	Env     map[string]string `json:"env,omitempty"`
	Files   map[string]string `json:"files,omitempty"`
}

type BuildContext struct {
	Agent    store.Agent
	Provider store.Provider
	Profile  *store.Profile
	Model    string
	Args     []string
	Options  LaunchOptions
}

type Builder func(BuildContext) (LaunchPlan, func(), error)

var builders = map[string]Builder{
	"claude":     buildClaudePlan,
	"codex":      buildCodexPlan,
	"gemini":     buildGeminiPlan,
	"openai_env": buildOpenAIEnvPlan,
	"hermes":     buildOpenAIEnvPlan,
	"openclaw":   buildOpenAIEnvPlan,
}

func Register(name string, builder Builder) {
	builders[name] = builder
}

func BuilderNames() []string {
	names := make([]string, 0, len(builders))
	for name := range builders {
		names = append(names, name)
	}
	return names
}

func BuildPlan(agent store.Agent, provider store.Provider, profile *store.Profile, opts LaunchOptions) (LaunchPlan, func(), error) {
	args := append(defaultArgs(profile), opts.Args...)
	model := provider.DefaultModel
	if profile != nil && profile.Model != "" {
		model = profile.Model
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return LaunchPlan{}, nil, fmt.Errorf("provider %s has no default model; set provider.default_model or profile.model", provider.Slug)
	}

	builder, ok := builders[agent.Adapter]
	if !ok {
		return LaunchPlan{}, nil, fmt.Errorf("unsupported adapter: %s", agent.Adapter)
	}
	return builder(BuildContext{Agent: agent, Provider: provider, Profile: profile, Model: model, Args: args, Options: opts})
}

func Execute(plan LaunchPlan, cleanup func(), opts LaunchOptions) error {
	defer cleanup()
	if opts.DryRun {
		return nil
	}
	if opts.Terminal != "" && opts.Terminal != "current" {
		return launchTerminal(opts.Terminal, withEnvPrefix(plan.Command, plan.Env), plan.CWD)
	}
	cmd := exec.Command("/bin/sh", "-lc", plan.Command)
	if plan.CWD != "" {
		cmd.Dir = plan.CWD
	}
	cmd.Env = os.Environ()
	for key, value := range plan.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func withEnvPrefix(command string, env map[string]string) string {
	if len(env) == 0 {
		return command
	}
	parts := make([]string, 0, len(env)+1)
	for key, value := range env {
		parts = append(parts, key+"="+shellQuote(value))
	}
	parts = append(parts, command)
	return strings.Join(parts, " ")
}

func buildClaudePlan(ctx BuildContext) (LaunchPlan, func(), error) {
	agent := ctx.Agent
	provider := ctx.Provider
	profile := ctx.Profile
	model := ctx.Model
	args := ctx.Args
	opts := ctx.Options
	settings := map[string]interface{}{
		"env": map[string]string{
			"ANTHROPIC_AUTH_TOKEN": provider.APIKey,
			"ANTHROPIC_API_KEY":    provider.APIKey,
			"ANTHROPIC_BASE_URL":   provider.BaseURL,
		},
	}
	if model != "" {
		env := settings["env"].(map[string]string)
		env["ANTHROPIC_MODEL"] = model
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
	}
	if extraEnv, ok := provider.Capabilities["claude_extra_env"].(map[string]interface{}); ok {
		env := settings["env"].(map[string]string)
		for k, v := range extraEnv {
			if s, ok := v.(string); ok {
				env[k] = s
			}
		}
	}
	mergeOverrides(settings, profile)

	path, cleanup, err := writeTempJSON("aisw-claude-", ".json", settings)
	if err != nil {
		return LaunchPlan{}, nil, err
	}
	cmd := joinCommand(append([]string{agent.Binary, "--settings", path}, args...))
	return LaunchPlan{Command: cmd, CWD: opts.CWD, Files: map[string]string{"settings": path}}, cleanup, nil
}

func buildCodexPlan(ctx BuildContext) (LaunchPlan, func(), error) {
	agent := ctx.Agent
	provider := ctx.Provider
	profile := ctx.Profile
	model := ctx.Model
	args := ctx.Args
	opts := ctx.Options
	dir, err := os.MkdirTemp("", "aisw-codex-")
	if err != nil {
		return LaunchPlan{}, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	providerKey := sanitizeProviderKey(provider.Slug)
	authMode := "requires_openai_auth"
	if mode, ok := provider.Capabilities["codex_auth_mode"].(string); ok {
		authMode = mode
	}

	var config string
	if authMode == "experimental_bearer_token" {
		contextWindow := ""
		if cw := getIntCapability(provider.Capabilities, "codex_model_context_window"); cw > 0 {
			contextWindow = fmt.Sprintf("\nmodel_context_window = %d", cw)
		}
		config = fmt.Sprintf("model = %q\nmodel_provider = %q%s\n\n[model_providers.%s]\nname = %q\nbase_url = %q\nexperimental_bearer_token = %q\nwire_api = %q\n", model, providerKey, contextWindow, providerKey, providerKey, provider.BaseURL, provider.APIKey, codexWireAPI(provider.APIProtocol))
	} else {
		config = fmt.Sprintf("model_provider = %q\nmodel = %q\n\n[model_providers.%s]\nname = %q\nbase_url = %q\nwire_api = %q\nrequires_openai_auth = true\n", providerKey, model, providerKey, providerKey, provider.BaseURL, codexWireAPI(provider.APIProtocol))
	}
	if profile != nil && len(profile.ConfigOverrides) > 0 {
		if extra, ok := profile.ConfigOverrides["codex_config_append"].(string); ok && strings.TrimSpace(extra) != "" {
			config += "\n" + extra + "\n"
		}
	}

	configPath := filepath.Join(dir, "config.toml")
	if err := safefile.Write(configPath, []byte(config), 0o600); err != nil {
		cleanup()
		return LaunchPlan{}, nil, err
	}

	cmd := joinCommand(append([]string{agent.Binary}, args...))
	plan := LaunchPlan{
		Command: cmd,
		CWD:     opts.CWD,
		Env:     map[string]string{"CODEX_HOME": dir},
		Files:   map[string]string{"codex_home": dir, "config": configPath},
	}

	if authMode != "experimental_bearer_token" {
		authPath := filepath.Join(dir, "auth.json")
		auth := map[string]string{"OPENAI_API_KEY": provider.APIKey}
		bytes, _ := json.MarshalIndent(auth, "", "  ")
		if err := safefile.Write(authPath, bytes, 0o600); err != nil {
			cleanup()
			return LaunchPlan{}, nil, err
		}
		plan.Files["auth"] = authPath
	}

	return plan, cleanup, nil
}

func buildGeminiPlan(ctx BuildContext) (LaunchPlan, func(), error) {
	return buildEnvPlan(ctx.Agent.Binary, map[string]string{
		"GEMINI_API_KEY":         ctx.Provider.APIKey,
		"GOOGLE_GEMINI_BASE_URL": ctx.Provider.BaseURL,
	}, ctx.Profile, ctx.Args, ctx.Options), func() {}, nil
}

func buildOpenAIEnvPlan(ctx BuildContext) (LaunchPlan, func(), error) {
	return buildEnvPlan(ctx.Agent.Binary, map[string]string{
		"OPENAI_API_KEY":  ctx.Provider.APIKey,
		"OPENAI_BASE_URL": ctx.Provider.BaseURL,
	}, ctx.Profile, ctx.Args, ctx.Options), func() {}, nil
}

func buildEnvPlan(binary string, env map[string]string, profile *store.Profile, args []string, opts LaunchOptions) LaunchPlan {
	for key, value := range envOverrides(profile) {
		env[key] = value
	}
	return LaunchPlan{Command: joinCommand(append([]string{binary}, args...)), CWD: opts.CWD, Env: env}
}

func defaultArgs(profile *store.Profile) []string {
	if profile == nil || strings.TrimSpace(profile.DefaultArgs) == "" {
		return nil
	}
	return strings.Fields(profile.DefaultArgs)
}

func envOverrides(profile *store.Profile) map[string]string {
	if profile == nil || profile.EnvOverrides == nil {
		return map[string]string{}
	}
	return profile.EnvOverrides
}

func mergeOverrides(settings map[string]interface{}, profile *store.Profile) {
	if profile == nil {
		return
	}
	if env := envOverrides(profile); len(env) > 0 {
		settingsEnv, _ := settings["env"].(map[string]string)
		for key, value := range env {
			settingsEnv[key] = value
		}
	}
	for key, value := range profile.ConfigOverrides {
		if key == "codex_config_append" {
			continue
		}
		settings[key] = value
	}
}

func writeTempJSON(prefix, suffix string, value interface{}) (string, func(), error) {
	file, err := os.CreateTemp("", prefix+"*"+suffix)
	if err != nil {
		return "", nil, err
	}
	path := file.Name()
	cleanup := func() { _ = os.Remove(path) }
	if err := os.Chmod(path, 0o600); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, err
	}
	bytes, _ := json.MarshalIndent(value, "", "  ")
	if _, err := file.Write(bytes); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

func codexWireAPI(protocol string) string {
	switch protocol {
	case "openai_responses":
		return "responses"
	default:
		return "chat"
	}
}

func getIntCapability(capabilities map[string]interface{}, key string) int {
	if v, ok := capabilities[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

func sanitizeProviderKey(value string) string {
	value = strings.Trim(strings.ToLower(value), "-")
	var b strings.Builder
	lastDash := false
	for _, ch := range value {
		valid := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if valid {
			b.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "provider"
	}
	return result
}

func joinCommand(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		quoted = append(quoted, shellQuote(part))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && !strings.ContainsRune("-_=.,/:@%+", r)
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func launchTerminal(terminal, command, cwd string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("terminal launch is currently implemented only for macOS")
	}
	full := command
	if cwd != "" {
		full = "cd " + shellQuote(cwd) + " && " + command
	}
	switch terminal {
	case "ghostty":
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/zsh"
		}
		args := []string{"-na", "Ghostty", "--args", "--quit-after-last-window-closed=true"}
		if cwd != "" {
			args = append(args, "--working-directory="+cwd)
		}
		args = append(args, "-e", shell, "-l", "-c", command)
		return exec.Command("open", args...).Run()
	case "terminal":
		escaped := strings.ReplaceAll(strings.ReplaceAll(full, `\`, `\\`), `"`, `\"`)
		script := fmt.Sprintf("tell application \"Terminal\" to activate\ntell application \"Terminal\" to do script \"%s\"", escaped)
		return exec.Command("osascript", "-e", script).Run()
	default:
		return fmt.Errorf("unsupported terminal: %s", terminal)
	}
}

// PersistDefault writes the provider/profile configuration into the agent's
// persistent config file (e.g. ~/.claude/settings.json) so that running the
// agent binary directly (without aisw) uses the default provider.
func PersistDefault(agent store.Agent, provider store.Provider, profile *store.Profile) error {
	switch agent.Adapter {
	case "claude":
		return persistClaudeDefault(provider, profile)
	default:
		// Other adapters do not yet support persistent default configs.
		return nil
	}
}

func persistClaudeDefault(provider store.Provider, profile *store.Profile) error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}

	settings := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &settings)
	}

	settingsEnv := map[string]string{
		"ANTHROPIC_AUTH_TOKEN": provider.APIKey,
		"ANTHROPIC_API_KEY":    provider.APIKey,
		"ANTHROPIC_BASE_URL":   provider.BaseURL,
	}
	model := provider.DefaultModel
	if profile != nil && profile.Model != "" {
		model = profile.Model
	}
	if model != "" {
		settingsEnv["ANTHROPIC_MODEL"] = model
		settingsEnv["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
		settingsEnv["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
		settingsEnv["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
	}
	if extraEnv, ok := provider.Capabilities["claude_extra_env"].(map[string]interface{}); ok {
		for k, v := range extraEnv {
			if s, ok := v.(string); ok {
				settingsEnv[k] = s
			}
		}
	}
	settings["env"] = settingsEnv

	mergeOverrides(settings, profile)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return safefile.Write(path, data, 0o600)
}

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}
