// cc-switch-mini - Minimal LLM provider proxy with TUI select, templates, and terminal launcher.
//
// Usage:
//   cc-switch-mini               # interactive TUI select mode
//   cc-switch-mini proxy         # start proxy server
//   cc-switch-mini templates     # browse & add from built-in templates
//   cc-switch-mini launch -provider MiniMax -app codex -terminal ghostty
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ── Config ─────────────────────────────────────────────────────

type ProviderConfig struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type Config struct {
	ListenAddr string           `json:"listen_addr"`
	ListenPort int              `json:"listen_port"`
	Providers  []ProviderConfig `json:"providers"`
}

type ProviderTemplate struct {
	Name        string
	BaseURL     string
	Description string
}

func allTemplates() []ProviderTemplate {
	return []ProviderTemplate{
		{"Anthropic 官方", "https://api.anthropic.com", "Claude 官方 API，需要海外支付"},
		{"OpenAI 官方", "https://api.openai.com", "GPT 官方 API，需要海外支付"},
		{"MiniMax", "https://api.minimax.chat", "国内可用，Claude/Codex/Gemini 全系中转"},
		{"硅基流动 SiliconFlow", "https://api.siliconflow.cn", "国内可用，支持 Claude/Gemini/DeepSeek/Qwen"},
		{"火山方舟 VolcEngine", "https://ark.cn-beijing.volces.com/api/v3", "字节跳动，豆包/DeepSeek/GLM"},
		{"胜算云", "https://api.shengsuanyun.com", "Claude/GPT/Gemini 全系中转，企业级 SLA"},
		{"CrazyRouter", "https://api.crazyrouter.com", "聚合 300+ 模型，低至 55%"},
		{"PackyCode", "https://api.packyapi.com", "Claude Code/Codex/Gemini 中转"},
		{"AICodeMirror", "https://api.aicodemirror.com", "Claude Code 低至 3.8 折"},
		{"Groq", "https://api.groq.com/openai/v1", "超低延迟推理"},
		{"DeepSeek 官方", "https://api.deepseek.com", "DeepSeek 官方 API，极低成本"},
		{"DMXAPI", "https://api.dmxapi.cn", "全系模型 6.8 折"},
		{"PatewayAI", "https://api.pateway.ai", "100% 官方直供"},
		{"LemonData", "https://api.lemondata.cc", "300+ 模型，30%-70% 官方定价"},
	}
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cc-switch-mini.json")
}

func defaultConfig() Config {
	return Config{
		ListenAddr: "127.0.0.1",
		ListenPort: 15721,
		Providers: []ProviderConfig{
			{Name: "MiniMax", BaseURL: "https://api.minimax.chat", APIKey: "sk-your-minimax-key"},
			{Name: "OpenAI", BaseURL: "https://api.openai.com", APIKey: "sk-your-openai-key"},
		},
	}
}

func loadConfig(path string) (Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := defaultConfig()
		data, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(path, data, 0644); err != nil {
			return cfg, err
		}
		log.Printf("[config] wrote default to %s", path)
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func saveConfig(path string, cfg Config) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, data, 0644)
}

func findProvider(cfg Config, name string) *ProviderConfig {
	lower := strings.ToLower(name)
	for i := range cfg.Providers {
		if strings.ToLower(cfg.Providers[i].Name) == lower {
			return &cfg.Providers[i]
		}
	}
	return nil
}

func providerIndex(cfg Config, name string) int {
	lower := strings.ToLower(name)
	for i, p := range cfg.Providers {
		if strings.ToLower(p.Name) == lower {
			return i
		}
	}
	return -1
}

// ── Circuit Breaker ────────────────────────────────────────────

const (
	stateClosed   int32 = 0
	stateOpen     int32 = 1
	stateHalfOpen int32 = 2
)

type circuitBreaker struct {
	state                int32
	consecutiveFailures  int32
	consecutiveSuccesses int32
	lastOpenedAt         time.Time
	mu                   sync.Mutex
	failureThreshold     int32
	successThreshold     int32
	timeoutSecs          int64
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{
		failureThreshold: 4,
		successThreshold: 2,
		timeoutSecs:      60,
	}
}

func (cb *circuitBreaker) isAvailable() bool {
	s := atomic.LoadInt32(&cb.state)
	if s == stateClosed || s == stateHalfOpen {
		return true
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if time.Since(cb.lastOpenedAt).Seconds() >= float64(cb.timeoutSecs) {
		atomic.StoreInt32(&cb.state, stateHalfOpen)
		log.Printf("[breaker] Open → HalfOpen (timeout)")
		return true
	}
	return false
}

func (cb *circuitBreaker) recordSuccess() {
	atomic.StoreInt32(&cb.consecutiveFailures, 0)
	if atomic.LoadInt32(&cb.state) == stateHalfOpen {
		n := atomic.AddInt32(&cb.consecutiveSuccesses, 1)
		if n >= cb.successThreshold {
			atomic.StoreInt32(&cb.state, stateClosed)
			atomic.StoreInt32(&cb.consecutiveSuccesses, 0)
			log.Printf("[breaker] HalfOpen → Closed (recovered)")
		}
	}
}

func (cb *circuitBreaker) recordFailure() {
	s := atomic.LoadInt32(&cb.state)
	if s == stateClosed {
		n := atomic.AddInt32(&cb.consecutiveFailures, 1)
		if n >= cb.failureThreshold {
			cb.mu.Lock()
			cb.lastOpenedAt = time.Now()
			cb.mu.Unlock()
			atomic.StoreInt32(&cb.state, stateOpen)
			log.Printf("[breaker] Closed → Open (%d failures)", n)
		}
	} else if s == stateHalfOpen {
		cb.mu.Lock()
		cb.lastOpenedAt = time.Now()
		cb.mu.Unlock()
		atomic.StoreInt32(&cb.state, stateOpen)
		atomic.StoreInt32(&cb.consecutiveSuccesses, 0)
		log.Printf("[breaker] HalfOpen → Open (probe failed)")
	}
}

// ── App State ──────────────────────────────────────────────────

type appState struct {
	config     Config
	breakers   map[string]*circuitBreaker
	breakersMu sync.Mutex
	client     *http.Client
}

func (s *appState) getBreaker(name string) *circuitBreaker {
	s.breakersMu.Lock()
	defer s.breakersMu.Unlock()
	if b, ok := s.breakers[name]; ok {
		return b
	}
	b := newCircuitBreaker()
	s.breakers[name] = b
	return b
}

// ── Proxy Handler ──────────────────────────────────────────────

func (s *appState) proxyHandler(w http.ResponseWriter, r *http.Request) {
	pathAndQuery := r.URL.Path
	if r.URL.RawQuery != "" {
		pathAndQuery += "?" + r.URL.RawQuery
	}
	defer r.Body.Close()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body"}`, http.StatusBadRequest)
		return
	}
	total := len(s.config.Providers)
	for attempt := 0; attempt < total; attempt++ {
		p := s.config.Providers[attempt]
		breaker := s.getBreaker(p.Name)
		if !breaker.isAvailable() {
			log.Printf("[%d/%d] %s is OPEN, skipping...", attempt+1, total, p.Name)
			continue
		}
		log.Printf("[%d/%d] %s %s → %s", attempt+1, total, r.Method, pathAndQuery, p.Name)
		upstreamURL := p.BaseURL + pathAndQuery
		resp, err := s.forward(r, upstreamURL, p.APIKey, bodyBytes)
		if err != nil {
			breaker.recordFailure()
			log.Printf("[%d/%d] %s FAILED: %v", attempt+1, total, p.Name, err)
			continue
		}
		breaker.recordSuccess()
		log.Printf("[%d/%d] %s returned %d", attempt+1, total, p.Name, resp.StatusCode)
		for k, vs := range resp.Header {
			if k == "Transfer-Encoding" {
				continue
			}
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		resp.Body.Close()
		return
	}
	http.Error(w, fmt.Sprintf(`{"error":"all %d providers failed"}`, total), http.StatusBadGateway)
}

func (s *appState) forward(original *http.Request, upstreamURL, apiKey string, body []byte) (*http.Response, error) {
	u, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(original.Context(), original.Method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range original.Header {
		low := http.CanonicalHeaderKey(k)
		if low == "Host" || low == "Authorization" || low == "Content-Length" {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		req.Body = io.NopCloser(strings.NewReader(string(body)))
		req.ContentLength = int64(len(body))
	}
	return s.client.Do(req)
}

// ── Terminal Launcher ──────────────────────────────────────────

func runProxyServer(cfg Config) error {
	state := &appState{
		config:   cfg,
		breakers: make(map[string]*circuitBreaker),
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/", state.proxyHandler)
	listenAddr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort)
	log.Printf("cc-switch-mini proxy on %s", listenAddr)
	for i, p := range cfg.Providers {
		log.Printf("  Provider %d: %s -> %s", i+1, p.Name, p.BaseURL)
	}
	return http.ListenAndServe(listenAddr, mux)
}

func launchTerminal(terminal, command, cwd string, env map[string]string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("terminal launch only supported on macOS")
	}
	var envStr string
	for k, v := range env {
		envStr += fmt.Sprintf(`%s="%s" `, k, v)
	}
	fullCmd := envStr + command
	switch terminal {
	case "terminal":
		return launchMacOSTerminal(fullCmd, cwd)
	case "iterm":
		return launchITerm(fullCmd, cwd)
	case "ghostty":
		return launchGhostty(fullCmd, cwd)
	case "kitty":
		return launchKitty(fullCmd, cwd)
	default:
		return fmt.Errorf("unsupported terminal: %s", terminal)
	}
}

func launchMacOSTerminal(command, cwd string) error {
	full := buildShellCmd(command, cwd)
	esc := strings.ReplaceAll(strings.ReplaceAll(full, `\`, `\\`), `"`, `\"`)
	script := fmt.Sprintf(`tell application "Terminal" to activate
tell application "Terminal" to do script "%s"`, esc)
	return exec.Command("osascript", "-e", script).Run()
}

func launchITerm(command, cwd string) error {
	full := buildShellCmd(command, cwd)
	esc := strings.ReplaceAll(strings.ReplaceAll(full, `\`, `\\`), `"`, `\"`)
	script := fmt.Sprintf(`tell application "iTerm" to activate
tell application "iTerm"
    create window with default profile
    tell current session of current window
        write text "%s"
    end tell
end tell`, esc)
	return exec.Command("osascript", "-e", script).Run()
}

func launchGhostty(command, cwd string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	args := []string{"-na", "Ghostty", "--args", "--quit-after-last-window-closed=true"}
	if cwd != "" {
		args = append(args, fmt.Sprintf("--working-directory=%s", cwd))
	}
	args = append(args, "-e", shell, "-l", "-c", command)
	return exec.Command("open", args...).Run()
}

func launchKitty(command, cwd string) error {
	full := buildShellCmd(command, cwd)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	return exec.Command("open", "-na", "kitty", "--args", "-e", shell, "-l", "-c", full).Run()
}

func buildShellCmd(command, cwd string) string {
	if cwd != "" {
		return fmt.Sprintf(`cd "%s" && %s`, cwd, command)
	}
	return command
}

func doLaunch(provider ProviderConfig, app, terminal string, extraArgs []string) error {
	fmt.Printf("\n  App:      %s\n", app)
	fmt.Printf("  Provider: %s\n", provider.Name)
	fmt.Printf("  URL:      %s\n", provider.BaseURL)
	fmt.Printf("  Terminal: %s\n\n", terminal)

	cmd := app
	if len(extraArgs) > 0 {
		cmd += " " + strings.Join(extraArgs, " ")
	}

	var envMap map[string]string
	switch app {
	case "claude":
		envMap = map[string]string{"ANTHROPIC_API_KEY": provider.APIKey, "ANTHROPIC_BASE_URL": provider.BaseURL}
	case "codex":
		envMap = map[string]string{"OPENAI_API_KEY": provider.APIKey, "OPENAI_BASE_URL": provider.BaseURL}
	case "gemini":
		envMap = map[string]string{"GEMINI_API_KEY": provider.APIKey}
	default:
		return fmt.Errorf("unsupported app: %s", app)
	}

	if err := launchTerminal(terminal, cmd, "", envMap); err != nil {
		return err
	}
	fmt.Printf("Done! %s started with provider %q\n", app, provider.Name)
	return nil
}

// ── TUI Select ─────────────────────────────────────────────────

func tuiSelect(items []string, prompt string) int {
	if len(items) == 0 {
		return -1
	}
	selected := 0
	for {
		fmt.Fprint(os.Stderr, "\033[H\033[2J")
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", prompt)
		for i, item := range items {
			if i == selected {
				fmt.Fprintf(os.Stderr, "  \033[7m ▶ %s \033[0m\n", item)
			} else {
				fmt.Fprintf(os.Stderr, "    %s\n", item)
			}
		}
		fmt.Fprintf(os.Stderr, "\n  ↑↓ 移动  Enter 确认  q 退出\n")
		buf := make([]byte, 3)
		n, _ := os.Stdin.Read(buf)
		if n == 1 {
			if buf[0] == 'q' || buf[0] == 27 {
				return -1
			}
			if buf[0] == '\r' || buf[0] == '\n' {
				return selected
			}
		}
		if n == 3 {
			if buf[0] == 27 && buf[1] == 91 {
				switch buf[2] {
				case 65: // up
					if selected > 0 {
						selected--
					}
				case 66: // down
					if selected < len(items)-1 {
						selected++
					}
				}
			}
		}
	}
}

func tuiInput(prompt string) string {
	fmt.Fprintf(os.Stderr, "  %s: ", prompt)
	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

func tuiConfirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "  %s (y/n): ", prompt)
	var ans string
	fmt.Scanln(&ans)
	return strings.ToLower(strings.TrimSpace(ans)) == "y"
}

func runTUI(cfg Config, configPath string) error {
	apps := []string{"claude", "codex", "gemini"}
	terminals := []string{"ghostty", "kitty", "iterm", "terminal"}

	if len(cfg.Providers) == 0 {
		fmt.Println("还没有配置 Provider，先添加一个\n")
		return runTemplates(cfg, configPath)
	}

	// Step 1: pick app
	app := apps[tuiSelect(apps, "选择 AI 工具")]
	if app == "" {
		return nil
	}

	// Step 2: pick provider
	names := make([]string, len(cfg.Providers))
	for i, p := range cfg.Providers {
		names[i] = fmt.Sprintf("%s  →  %s", p.Name, p.BaseURL)
	}
	names = append(names, fmt.Sprintf("➕ 新增 Provider... (共 %d 个预设)", len(allTemplates())))

	pIdx := tuiSelect(names, fmt.Sprintf("选择 Provider (%s)", app))
	if pIdx < 0 {
		return nil
	}

	var provider ProviderConfig
	if pIdx >= len(cfg.Providers) {
		return runTemplates(cfg, configPath)
	} else {
		provider = cfg.Providers[pIdx]
	}

	// Step 3: pick terminal
	term := terminals[tuiSelect(terminals, "选择终端")]
	if term == "" {
		return nil
	}

	// Step 4: extra args
	args := tuiInput("额外参数 (可选，如 --resume abc123)")

	var extraArgs []string
	if args != "" {
		extraArgs = strings.Fields(args)
	}

	return doLaunch(provider, app, term, extraArgs)
}

func runTemplates(cfg Config, configPath string) error {
	templates := allTemplates()
	items := make([]string, len(templates))
	for i, t := range templates {
		items[i] = fmt.Sprintf("%s  —  %s", t.Name, t.Description)
	}

	idx := tuiSelect(items, "选择 Provider 模板")
	if idx < 0 {
		return nil
	}
	t := templates[idx]

	key := tuiInput(fmt.Sprintf("输入 %s 的 API Key", t.Name))
	if key == "" {
		fmt.Println("API Key 不能为空")
		return nil
	}

	existing := findProvider(cfg, t.Name)
	if existing != nil {
		fmt.Printf("Provider '%s' 已存在，更新 API Key...\n", t.Name)
		cfg.Providers = append(cfg.Providers[:providerIndex(cfg, t.Name)], cfg.Providers[providerIndex(cfg, t.Name)+1:]...)
	}
	cfg.Providers = append(cfg.Providers, ProviderConfig{Name: t.Name, BaseURL: t.BaseURL, APIKey: key})
	saveConfig(configPath, cfg)
	fmt.Printf("✅ 已添加 Provider: %s\n", t.Name)

	if tuiConfirm("是否立即启动?") {
		return doLaunch(ProviderConfig{Name: t.Name, BaseURL: t.BaseURL, APIKey: key}, "claude", "ghostty", nil)
	}
	return nil
}

// ── Main ───────────────────────────────────────────────────────

func main() {
	proxyCmd := flag.NewFlagSet("proxy", flag.ExitOnError)
	proxyAddr := proxyCmd.String("addr", "", "Listen address")
	proxyPort := proxyCmd.Int("port", 0, "Listen port")
	proxyConfig := proxyCmd.String("config", "", "Config file path")

	launchCmd := flag.NewFlagSet("launch", flag.ExitOnError)
	launchProvider := launchCmd.String("provider", "", "Provider name")
	launchApp := launchCmd.String("app", "claude", "App (claude/codex/gemini)")
	launchTerminal := launchCmd.String("terminal", "ghostty", "Terminal")
	launchConfig := launchCmd.String("config", "", "Config file path")

	loadPath := func(cf string) (Config, string) {
		p := cf
		if p == "" {
			p = defaultConfigPath()
		}
		cfg, err := loadConfig(p)
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		return cfg, p
	}

	if len(os.Args) < 2 {
		cfg, p := loadPath("")
		if err := runTUI(cfg, p); err != nil {
			log.Fatal(err)
		}
		return
	}

	switch os.Args[1] {
	case "proxy":
		proxyCmd.Parse(os.Args[2:])
		cfg, _ := loadPath(*proxyConfig)
		if *proxyAddr != "" {
			cfg.ListenAddr = *proxyAddr
		}
		if *proxyPort != 0 {
			cfg.ListenPort = *proxyPort
		}
		log.Fatal(runProxyServer(cfg))

	case "launch":
		launchCmd.Parse(os.Args[2:])
		if *launchProvider == "" {
			log.Fatal("-provider is required")
		}
		cfg, _ := loadPath(*launchConfig)
		p := findProvider(cfg, *launchProvider)
		if p == nil {
			log.Fatalf("provider %q not found", *launchProvider)
		}
		if err := doLaunch(*p, *launchApp, *launchTerminal, launchCmd.Args()); err != nil {
			log.Fatal(err)
		}

	case "templates":
		cfg, p := loadPath("")
		if err := runTemplates(cfg, p); err != nil {
			log.Fatal(err)
		}

	default:
		log.Fatalf("unknown command: %s (use: proxy | launch | templates | no args for TUI)", os.Args[1])
	}
}
