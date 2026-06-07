package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/ui"
	"github.com/spf13/cobra"
	"github.com/variableway/innate-aiswitcher/internal/adapter"
	"github.com/variableway/innate-aiswitcher/internal/configfile"
	"github.com/variableway/innate-aiswitcher/internal/httpcheck"
	"github.com/variableway/innate-aiswitcher/internal/projectconfig"
	"github.com/variableway/innate-aiswitcher/internal/store"
	"github.com/variableway/innate-aiswitcher/internal/templates"
	"github.com/variableway/innate-aiswitcher/internal/tui"
)

var defaultAdminUIDistFS = ui.DistDirFS

type Options struct {
	DataDir         string
	EnableAdminUI   bool
	ShowAdminBanner bool
}

type PBGetter func() (*pocketbase.PocketBase, error)

func NewCLI() *cobra.Command {
	opts := Options{DataDir: defaultDataDir()}
	var pb *pocketbase.PocketBase
	getPB := func() (*pocketbase.PocketBase, error) {
		if pb != nil {
			return pb, nil
		}
		pb = NewWithOptions(opts)
		if err := pb.Bootstrap(); err != nil {
			return nil, err
		}
		return pb, nil
	}

	cmd := &cobra.Command{
		Use:          "aisw",
		Short:        "AI provider switcher for local coding agents",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			return tui.Run(store.New(pb))
		},
	}
	cmd.PersistentFlags().StringVar(&opts.DataDir, "dir", opts.DataDir, "the PocketBase data directory")
	cmd.PersistentFlags().BoolVar(&opts.EnableAdminUI, "admin-ui", false, "enable the PocketBase admin UI at /_")
	cmd.PersistentFlags().BoolVar(&opts.ShowAdminBanner, "show-admin-banner", false, "show the PocketBase startup banner and admin install URL")
	cmd.AddCommand(
		providerCommand(getPB),
		profileCommand(getPB),
		startCommand(getPB),
		testCommand(getPB),
		configCommand(getPB),
		initCommand(getPB),
		serveCommand(getPB, &opts),
	)
	return cmd
}

func New() *pocketbase.PocketBase {
	return NewWithOptions(optionsFromArgs(os.Args[1:]))
}

func NewWithOptions(opts Options) *pocketbase.PocketBase {
	if opts.EnableAdminUI {
		ui.DistDirFS = defaultAdminUIDistFS
	} else {
		ui.DistDirFS = nil
	}

	pb := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:  firstNonEmpty(opts.DataDir, defaultDataDir()),
		HideStartBanner: !opts.ShowAdminBanner,
	})
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
	pb.RootCmd.Use = "aisw"
	pb.RootCmd.Short = "AI provider switcher for local coding agents"
	pb.RootCmd.SilenceUsage = true
	pb.RootCmd.PersistentFlags().Bool("admin-ui", opts.EnableAdminUI, "enable the PocketBase admin UI at /_")
	pb.RootCmd.PersistentFlags().Bool("show-admin-banner", opts.ShowAdminBanner, "show the PocketBase startup banner and admin install URL")
	pb.RootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return tui.Run(store.New(pb))
	}

	registerRoutes(pb, opts)
	getPB := func() (*pocketbase.PocketBase, error) { return pb, nil }
	pb.RootCmd.AddCommand(
		providerCommand(getPB),
		profileCommand(getPB),
		startCommand(getPB),
		testCommand(getPB),
		configCommand(getPB),
		initCommand(getPB),
	)
	return pb
}

func optionsFromArgs(args []string) Options {
	return Options{
		EnableAdminUI:   boolFlagEnabled(args, "--admin-ui"),
		ShowAdminBanner: boolFlagEnabled(args, "--show-admin-banner"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func boolFlagEnabled(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
		if strings.HasPrefix(arg, name+"=") {
			value, err := strconv.ParseBool(strings.TrimPrefix(arg, name+"="))
			return err == nil && value
		}
	}
	return false
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "pb_data"
	}
	return filepath.Join(home, ".innate-aiswitcher", "pb_data")
}

func registerRoutes(pb *pocketbase.PocketBase, opts Options) {
	pb.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !opts.ShowAdminBanner {
				e.InstallerFunc = nil
			}
			e.Router.GET("/api/aisw/health", func(e *core.RequestEvent) error {
				return e.JSON(http.StatusOK, map[string]any{"ok": true, "service": "innate-aiswitcher"})
			})
			e.Router.GET("/api/aisw/catalog", func(e *core.RequestEvent) error {
				s := store.New(pb)
				agents, _ := s.ListAgents()
				providers, _ := s.ListProviders()
				for i := range providers {
					providers[i].APIKey = ""
				}
				return e.JSON(http.StatusOK, map[string]any{"agents": agents, "providers": providers})
			})
			e.Router.GET("/api/aisw/providers/{slug}/models", func(e *core.RequestEvent) error {
				s := store.New(pb)
				provider, err := s.GetProvider(e.Request.PathValue("slug"))
				if err != nil || provider == nil {
					return e.JSON(http.StatusNotFound, map[string]any{"ok": false, "message": "provider not found"})
				}
				result, err := httpcheck.ListModels(e.Request.Context(), nil, *provider)
				status := http.StatusOK
				if err != nil || !result.OK {
					status = http.StatusBadGateway
				}
				return e.JSON(status, result)
			})
			e.Router.POST("/api/aisw/providers/{slug}/test", func(e *core.RequestEvent) error {
				s := store.New(pb)
				provider, err := s.GetProvider(e.Request.PathValue("slug"))
				if err != nil || provider == nil {
					return e.JSON(http.StatusNotFound, map[string]any{"ok": false, "message": "provider not found"})
				}

				var payload struct {
					Model string `json:"model"`
				}
				if e.Request.Body != nil {
					_ = json.NewDecoder(e.Request.Body).Decode(&payload)
				}

				result, err := httpcheck.CheckProvider(e.Request.Context(), nil, *provider, payload.Model)
				status := http.StatusOK
				if err != nil || !result.OK {
					status = http.StatusBadGateway
				}
				return e.JSON(status, result)
			})
			return e.Next()
		},
	})
}

func providerCommand(getPB PBGetter) *cobra.Command {
	cmd := &cobra.Command{Use: "provider", Short: "Manage shared LLM providers"}

	var add store.Provider
	var endpointFlags []string
	addCmd := &cobra.Command{
		Use:   "add SLUG",
		Short: "Add or update a shared provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			add.Slug = args[0]
			if add.APIKey == "" {
				if envName, _ := cmd.Flags().GetString("api-key-env"); envName != "" {
					add.APIKey = os.Getenv(envName)
				}
			}
			endpoints, err := parseEndpointFlags(endpointFlags)
			if err != nil {
				return err
			}
			if len(endpoints) > 0 {
				add.Endpoints = endpoints
			}
			provider, err := store.New(pb).UpsertProvider(add)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved provider %s (%s)\n", provider.Slug, provider.BaseURL)
			return nil
		},
	}
	addCmd.Flags().StringVar(&add.Name, "name", "", "display name")
	addCmd.Flags().StringVar(&add.BaseURL, "base-url", "", "provider base URL")
	addCmd.Flags().StringVar(&add.APIKey, "api-key", "", "API key")
	addCmd.Flags().String("api-key-env", "", "read API key from an environment variable")
	addCmd.Flags().StringVar(&add.APIProtocol, "protocol", "openai_chat", "anthropic|openai_chat|openai_responses|gemini_native|generic")
	addCmd.Flags().StringVar(&add.DefaultModel, "model", "", "default model")
	addCmd.Flags().StringArrayVar(&endpointFlags, "endpoint", nil, "endpoint override as key=path or key=https://host/path")
	addCmd.Flags().StringVar(&add.Notes, "notes", "", "notes")
	_ = addCmd.MarkFlagRequired("base-url")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List providers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			providers, err := store.New(pb).ListProviders()
			if err != nil {
				return err
			}
			for _, provider := range providers {
				keyState := "missing"
				if provider.APIKey != "" {
					keyState = "set"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\tkey=%s\n", provider.Slug, provider.Name, provider.APIProtocol, provider.DefaultModel, provider.BaseURL, keyState)
			}
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete SLUG",
		Short: "Delete a provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			return store.New(pb).DeleteProvider(args[0])
		},
	}

	presetsCmd := &cobra.Command{
		Use:   "presets",
		Short: "List built-in provider templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			presets, err := templates.ProviderPresets()
			if err != nil {
				return err
			}
			tui.PrintPresets(presets)
			return nil
		},
	}

	cmd.AddCommand(addCmd, listCmd, deleteCmd, presetsCmd)
	return cmd
}

func profileCommand(getPB PBGetter) *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Manage agent-specific provider profiles"}

	var profile store.Profile
	addCmd := &cobra.Command{
		Use:   "add SLUG",
		Short: "Add or update a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			profile.Slug = args[0]
			result, err := store.New(pb).UpsertProfile(profile)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved profile %s (%s -> %s)\n", result.Slug, result.AgentSlug, result.ProviderSlug)
			return nil
		},
	}
	addCmd.Flags().StringVar(&profile.Name, "name", "", "display name")
	addCmd.Flags().StringVar(&profile.AgentSlug, "agent", "", "agent slug")
	addCmd.Flags().StringVar(&profile.ProviderSlug, "provider", "", "provider slug")
	addCmd.Flags().StringVar(&profile.Model, "model", "", "model override")
	addCmd.Flags().StringVar(&profile.DefaultArgs, "args", "", "default native args")
	addCmd.Flags().BoolVar(&profile.IsDefault, "default", false, "mark as default")
	_ = addCmd.MarkFlagRequired("agent")
	_ = addCmd.MarkFlagRequired("provider")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			profiles, err := store.New(pb).ListProfiles()
			if err != nil {
				return err
			}
			for _, profile := range profiles {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", profile.Slug, profile.AgentSlug, profile.ProviderSlug, profile.Model)
			}
			return nil
		},
	}

	cmd.AddCommand(addCmd, listCmd)
	return cmd
}

func startCommand(getPB PBGetter) *cobra.Command {
	var dryRun bool
	var terminal string
	var cwd string
	var ignoreProject bool
	cmd := &cobra.Command{
		Use:   "start AGENT [PROVIDER_OR_PROFILE] -- [native args]",
		Short: "Start an agent with a session-only provider/profile",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			nativeArgs := []string{}
			if len(args) > 2 {
				nativeArgs = args[2:]
			}
			if cwd == "" {
				cwd, _ = os.Getwd()
			}

			agentSlug := args[0]
			selector := ""
			if len(args) > 1 {
				selector = args[1]
			}

			// Check .aiswrc unless --ignore-project is set
			if !ignoreProject && selector == "" {
				if cfg, foundDir, err := projectconfig.Find(cwd); err == nil && cfg != nil {
					if cfg.Profile != "" {
						selector = cfg.Profile
						fmt.Fprintf(cmd.OutOrStdout(), "using project profile %q from %s\n", cfg.Profile, foundDir)
					} else if cfg.Provider != "" {
						selector = cfg.Provider
						fmt.Fprintf(cmd.OutOrStdout(), "using project provider %q from %s\n", cfg.Provider, foundDir)
					}
					if cfg.Agent != "" && cfg.Agent != agentSlug {
						return fmt.Errorf(".aiswrc specifies agent %q but command asks for %q", cfg.Agent, agentSlug)
					}
				}
			}

			s := store.New(pb)
			agent, provider, profile, err := s.ResolveSelector(agentSlug, selector)
			if err != nil {
				return err
			}
			opts := adapter.LaunchOptions{CWD: cwd, Terminal: terminal, DryRun: dryRun, Args: nativeArgs}
			plan, cleanup, err := adapter.BuildPlan(*agent, *provider, profile, opts)
			if err != nil {
				return err
			}
			printPlan(cmd, plan)
			status := "started"
			launchErr := adapter.Execute(plan, cleanup, opts)
			if launchErr != nil {
				status = "failed"
			}
			profileID := ""
			if profile != nil {
				profileID = profile.ID
			}
			_ = s.SaveLaunchHistory(store.LaunchHistory{AgentID: agent.ID, ProviderID: provider.ID, ProfileID: profileID, CWD: cwd, Command: plan.Command, Terminal: terminal, Status: status, Error: errString(launchErr)})
			return launchErr
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the launch plan without starting the agent")
	cmd.Flags().StringVar(&terminal, "terminal", "current", "current|ghostty|terminal")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory")
	cmd.Flags().BoolVar(&ignoreProject, "ignore-project", false, "ignore .aiswrc project config")
	return cmd
}

func testCommand(getPB PBGetter) *cobra.Command {
	cmd := &cobra.Command{Use: "test", Short: "Test provider connectivity"}
	var model string
	providerCmd := &cobra.Command{
		Use:   "provider SLUG",
		Short: "Send a minimal model request using the provider API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			provider, err := store.New(pb).GetProvider(args[0])
			if err != nil || provider == nil {
				return fmt.Errorf("provider not found: %s", args[0])
			}
			result, err := httpcheck.CheckProvider(context.Background(), nil, *provider, model)
			fmt.Fprintln(cmd.OutOrStdout(), httpcheck.Format(result))
			if err != nil {
				return err
			}
			if !result.OK {
				return fmt.Errorf("provider test failed with status %d", result.StatusCode)
			}
			return nil
		},
	}
	providerCmd.Flags().StringVar(&model, "model", "", "model override for the test")

	modelsCmd := &cobra.Command{
		Use:   "models SLUG",
		Short: "List provider models using the provider API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			provider, err := store.New(pb).GetProvider(args[0])
			if err != nil || provider == nil {
				return fmt.Errorf("provider not found: %s", args[0])
			}
			result, err := httpcheck.ListModels(context.Background(), nil, *provider)
			fmt.Fprintln(cmd.OutOrStdout(), httpcheck.FormatModels(result))
			if err != nil {
				return err
			}
			if !result.OK {
				return fmt.Errorf("provider models failed with status %d", result.StatusCode)
			}
			return nil
		},
	}
	cmd.AddCommand(providerCmd, modelsCmd)
	return cmd
}

func configCommand(getPB PBGetter) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Import/export the shared config.toml mirror"}
	var path string
	var includeSecrets bool
	var backup bool
	var noBackup bool
	var backupPath string
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export SQLite data to config.toml",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			if path == "" {
				path = configfile.DefaultPath()
			}
			if err := configfile.Export(store.New(pb), path, includeSecrets); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %s\n", path)
			return nil
		},
	}
	exportCmd.Flags().StringVar(&path, "path", "", "config path")
	exportCmd.Flags().BoolVar(&includeSecrets, "include-secrets", false, "include API keys in the exported file")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Backup current SQLite config and import config.toml",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			if path == "" {
				path = configfile.DefaultPath()
			}
			s := store.New(pb)
			if backup && !noBackup {
				actualBackupPath, err := configfile.Backup(s, backupPath)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "backup %s\n", actualBackupPath)
			}
			if err := configfile.Import(s, path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "imported %s\n", path)
			return nil
		},
	}
	importCmd.Flags().StringVar(&path, "path", "", "config path")
	importCmd.Flags().BoolVar(&backup, "backup", true, "export a backup before importing")
	importCmd.Flags().BoolVar(&noBackup, "no-backup", false, "skip backup before importing")
	importCmd.Flags().StringVar(&backupPath, "backup-path", "", "backup path")

	templateCmd := &cobra.Command{
		Use:   "template",
		Short: "Write the embedded config template",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				path = configfile.DefaultPath()
			}
			if err := templates.WriteConfigExample(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote template %s\n", path)
			return nil
		},
	}
	templateCmd.Flags().StringVar(&path, "path", "", "config template path")

	cmd.AddCommand(exportCmd, importCmd, templateCmd)
	return cmd
}

func initCommand(getPB PBGetter) *cobra.Command {
	var profile string
	var agent string
	var provider string
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a .aiswrc project config in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			path := filepath.Join(cwd, ".aiswrc")
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%s already exists; use --force to overwrite", path)
			}

			cfg := projectconfig.ProjectConfig{
				Profile:  strings.TrimSpace(profile),
				Agent:    strings.TrimSpace(agent),
				Provider: strings.TrimSpace(provider),
			}
			if cfg.Profile == "" && cfg.Provider == "" {
				return fmt.Errorf("at least one of --profile or --provider is required")
			}
			if err := projectconfig.Write(cwd, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "default profile slug for this directory")
	cmd.Flags().StringVar(&agent, "agent", "", "agent slug (optional, for validation)")
	cmd.Flags().StringVar(&provider, "provider", "", "provider slug (optional, for direct provider binding)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing .aiswrc")
	return cmd
}

func serveCommand(getPB PBGetter, opts *Options) *cobra.Command {
	var allowedOrigins []string
	var httpAddr string
	var httpsAddr string
	cmd := &cobra.Command{
		Use:          "serve [domain(s)]",
		Args:         cobra.ArbitraryArgs,
		Short:        "Starts the local REST API server",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if httpAddr == "" {
					httpAddr = "0.0.0.0:80"
				}
				if httpsAddr == "" {
					httpsAddr = "0.0.0.0:443"
				}
			} else if httpAddr == "" {
				httpAddr = "127.0.0.1:8090"
			}

			pb, err := getPB()
			if err != nil {
				return err
			}
			err = apis.Serve(pb, apis.ServeConfig{
				HttpAddr:           httpAddr,
				HttpsAddr:          httpsAddr,
				ShowStartBanner:    opts.ShowAdminBanner,
				AllowedOrigins:     allowedOrigins,
				CertificateDomains: args,
			})
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		},
	}
	cmd.PersistentFlags().StringSliceVar(&allowedOrigins, "origins", []string{"*"}, "CORS allowed domain origins list")
	cmd.PersistentFlags().StringVar(&httpAddr, "http", "", "TCP address to listen for HTTP")
	cmd.PersistentFlags().StringVar(&httpsAddr, "https", "", "TCP address to listen for HTTPS")
	return cmd
}

func parseEndpointFlags(values []string) (map[string]string, error) {
	endpoints := map[string]string{}
	for _, value := range values {
		key, endpoint, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		endpoint = strings.TrimSpace(endpoint)
		if !ok || key == "" || endpoint == "" {
			return nil, fmt.Errorf("invalid endpoint override %q, expected key=path", value)
		}
		endpoints[key] = endpoint
	}
	return endpoints, nil
}

func formatStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	return strings.Join(parts, ",")
}

func printPlan(cmd *cobra.Command, plan adapter.LaunchPlan) {
	if bytes, err := json.MarshalIndent(plan, "", "  "); err == nil {
		fmt.Fprintln(cmd.OutOrStdout(), string(bytes))
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
