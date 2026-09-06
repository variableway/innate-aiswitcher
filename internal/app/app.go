package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/ui"
	"github.com/spf13/cobra"
	"github.com/variableway/innate-aiswitcher/internal/adapter"
	"github.com/variableway/innate-aiswitcher/internal/agentconfig"
	"github.com/variableway/innate-aiswitcher/internal/configfile"
	"github.com/variableway/innate-aiswitcher/internal/httpcheck"
	"github.com/variableway/innate-aiswitcher/internal/projectconfig"
	"github.com/variableway/innate-aiswitcher/internal/providerconfig"
	"github.com/variableway/innate-aiswitcher/internal/store"
	"github.com/variableway/innate-aiswitcher/internal/templates"
	"github.com/variableway/innate-aiswitcher/internal/terminal"
	"github.com/variableway/innate-aiswitcher/internal/tui"
	"github.com/variableway/innate-aiswitcher/internal/webui"
)

var defaultAdminUIDistFS = ui.DistDirFS

type Options struct {
	DataDir          string
	EnableAdminUI    bool
	ShowAdminBanner  bool
	DisableAccessLog bool
	InitConfig       string
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
		if err := maybeInitConfig(pb, opts.InitConfig); err != nil {
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
	cmd.PersistentFlags().StringVar(&opts.InitConfig, "init-config", "", "import config from a TOML file when the database is empty (defaults to "+configfile.InitConfigPath()+")")
	cmd.AddCommand(
		providerCommand(getPB),
		profileCommand(getPB),
		startCommand(getPB),
		testCommand(getPB),
		configCommand(getPB),
		serveCommand(getPB, &opts),
		webCommand(getPB, &opts),
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
			registerAccessLog(e, !opts.DisableAccessLog)
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

			// Provider CRUD
			registerProviderRoutes(e, pb)
			// Profile CRUD
			registerProfileRoutes(e, pb)
			// Agents (read-only)
			registerAgentRoutes(e, pb)
			// Presets
			registerPresetRoutes(e, pb)

			// Serve the web app (Vite + React SPA) with client-side routing
			// fallback. The mux resolves registered API routes first (most
			// specific pattern wins); only unmatched paths land here.
			webHandler, err := webui.Handler()
			if err == nil {
				e.Router.GET("/{path...}", func(ev *core.RequestEvent) error {
					path := ev.Request.URL.Path
					if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/_") {
						return apis.NewNotFoundError("", "not found")
					}
					webHandler.(http.Handler).ServeHTTP(ev.Response, ev.Request)
					return nil
				})
			}

			// Browser terminal sessions (WebSocket → local PTY).
			terminalHandler := terminal.Handler(terminal.Options{})
			e.Router.GET("/api/aisw/terminal", func(ev *core.RequestEvent) error {
				terminalHandler(ev.Response, ev.Request)
				return nil
			})

			// Agent configuration files (claude/codex/opencode) — view & edit.
			e.Router.GET("/api/aisw/agent-configs", func(ev *core.RequestEvent) error {
				agents, err := agentconfig.List()
				if err != nil {
					return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
				}
				return ev.JSON(http.StatusOK, agents)
			})
			// Config preview: project agent+provider+model through the real
			// launch pipeline and return the exact config files/env it would
			// use, without starting anything.
			e.Router.GET("/api/aisw/config-preview", func(ev *core.RequestEvent) error {
				s := store.New(pb)
				agent, err := s.GetAgent(ev.Request.URL.Query().Get("agent"))
				if err != nil || agent == nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "agent not found"})
				}
				provider, err := s.GetProvider(ev.Request.URL.Query().Get("provider"))
				if err != nil || provider == nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "provider not found"})
				}
				preview, err := adapter.Preview(*agent, *provider, ev.Request.URL.Query().Get("model"))
				if err != nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
				}
				return ev.JSON(http.StatusOK, preview)
			})
			e.Router.PUT("/api/aisw/agent-configs/{agent}/{name}", func(ev *core.RequestEvent) error {
				var payload struct {
					Content string `json:"content"`
				}
				if err := json.NewDecoder(ev.Request.Body).Decode(&payload); err != nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid JSON: " + err.Error()})
				}
				info, err := agentconfig.Write(ev.Request.PathValue("agent"), ev.Request.PathValue("name"), payload.Content)
				if err != nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
				}
				return ev.JSON(http.StatusOK, info)
			})

			return e.Next()
		},
	})
}

func registerProviderRoutes(e *core.ServeEvent, pb *pocketbase.PocketBase) {
	// List all providers (API key masked)
	e.Router.GET("/api/aisw/providers", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		providers, err := s.ListProviders()
		if err != nil {
			return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
		}
		for i := range providers {
			providers[i].APIKey = maskAPIKey(providers[i].APIKey)
		}
		return ev.JSON(http.StatusOK, providers)
	})

	// Get single provider
	e.Router.GET("/api/aisw/providers/{slug}", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		provider, err := s.GetProvider(ev.Request.PathValue("slug"))
		if err != nil || provider == nil {
			return ev.JSON(http.StatusNotFound, map[string]any{"ok": false, "message": "provider not found"})
		}
		provider.APIKey = maskAPIKey(provider.APIKey)
		return ev.JSON(http.StatusOK, provider)
	})

	// Create provider
	e.Router.POST("/api/aisw/providers", func(ev *core.RequestEvent) error {
		var input store.Provider
		if err := json.NewDecoder(ev.Request.Body).Decode(&input); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid JSON: " + err.Error()})
		}
		s := store.New(pb)
		provider, err := s.UpsertProvider(input)
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		provider.APIKey = maskAPIKey(provider.APIKey)
		return ev.JSON(http.StatusCreated, provider)
	})

	// Update provider
	e.Router.PUT("/api/aisw/providers/{slug}", func(ev *core.RequestEvent) error {
		var input store.Provider
		if err := json.NewDecoder(ev.Request.Body).Decode(&input); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid JSON: " + err.Error()})
		}
		input.Slug = ev.Request.PathValue("slug")
		s := store.New(pb)
		if input.APIKey == "" {
			existing, _ := s.GetProvider(input.Slug)
			if existing != nil {
				input.APIKey = existing.APIKey
			}
		}
		provider, err := s.UpsertProvider(input)
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		provider.APIKey = maskAPIKey(provider.APIKey)
		return ev.JSON(http.StatusOK, provider)
	})

	// Delete provider
	e.Router.DELETE("/api/aisw/providers/{slug}", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		if err := s.DeleteProvider(ev.Request.PathValue("slug")); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusOK, map[string]any{"ok": true})
	})

	// List provider models
	e.Router.GET("/api/aisw/providers/{slug}/models", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		provider, err := s.GetProvider(ev.Request.PathValue("slug"))
		if err != nil || provider == nil {
			return ev.JSON(http.StatusNotFound, map[string]any{"ok": false, "message": "provider not found"})
		}
		result, err2 := httpcheck.ListModels(ev.Request.Context(), nil, providerconfig.ResolveDefault(*provider))
		status := http.StatusOK
		if err2 != nil || !result.OK {
			status = http.StatusBadGateway
		}
		return ev.JSON(status, result)
	})

	// Test provider connectivity
	e.Router.POST("/api/aisw/providers/{slug}/test", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		provider, err := s.GetProvider(ev.Request.PathValue("slug"))
		if err != nil || provider == nil {
			return ev.JSON(http.StatusNotFound, map[string]any{"ok": false, "message": "provider not found"})
		}
		var payload struct {
			Model string `json:"model"`
		}
		if ev.Request.Body != nil {
			_ = json.NewDecoder(ev.Request.Body).Decode(&payload)
		}
		result, err2 := httpcheck.CheckProvider(ev.Request.Context(), nil, providerconfig.ResolveDefault(*provider), payload.Model)
		status := http.StatusOK
		if err2 != nil || !result.OK {
			status = http.StatusBadGateway
		}
		return ev.JSON(status, result)
	})

	// Add a model to a provider's configured model list (shares the API key)
	e.Router.POST("/api/aisw/providers/{slug}/models", func(ev *core.RequestEvent) error {
		var payload struct {
			Model   string `json:"model"`
			Default bool   `json:"default"`
		}
		if err := json.NewDecoder(ev.Request.Body).Decode(&payload); err != nil || strings.TrimSpace(payload.Model) == "" {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "model is required"})
		}
		s := store.New(pb)
		provider, err := s.AddModel(ev.Request.PathValue("slug"), payload.Model, payload.Default)
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		provider.APIKey = maskAPIKey(provider.APIKey)
		return ev.JSON(http.StatusCreated, provider)
	})

	// Remove a model from a provider's configured model list
	e.Router.DELETE("/api/aisw/providers/{slug}/models/{model}", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		provider, err := s.RemoveModel(ev.Request.PathValue("slug"), ev.Request.PathValue("model"))
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		provider.APIKey = maskAPIKey(provider.APIKey)
		return ev.JSON(http.StatusOK, provider)
	})
}

func registerProfileRoutes(e *core.ServeEvent, pb *pocketbase.PocketBase) {
	// List all profiles
	e.Router.GET("/api/aisw/profiles", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		profiles, err := s.ListProfiles()
		if err != nil {
			return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusOK, profiles)
	})

	// Create profile
	e.Router.POST("/api/aisw/profiles", func(ev *core.RequestEvent) error {
		var input store.Profile
		if err := json.NewDecoder(ev.Request.Body).Decode(&input); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid JSON: " + err.Error()})
		}
		s := store.New(pb)
		profile, err := s.UpsertProfile(input)
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusCreated, profile)
	})

	// Update profile
	e.Router.PUT("/api/aisw/profiles/{slug}", func(ev *core.RequestEvent) error {
		var input store.Profile
		if err := json.NewDecoder(ev.Request.Body).Decode(&input); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid JSON: " + err.Error()})
		}
		input.Slug = ev.Request.PathValue("slug")
		s := store.New(pb)
		profile, err := s.UpsertProfile(input)
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusOK, profile)
	})

	// Delete profile
	e.Router.DELETE("/api/aisw/profiles/{slug}", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		if err := s.DeleteProfile(ev.Request.PathValue("slug")); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusOK, map[string]any{"ok": true})
	})
}

func registerAgentRoutes(e *core.ServeEvent, pb *pocketbase.PocketBase) {
	e.Router.GET("/api/aisw/agents", func(ev *core.RequestEvent) error {
		s := store.New(pb)
		agents, err := s.ListAgents()
		if err != nil {
			return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusOK, agents)
	})
}

func registerPresetRoutes(e *core.ServeEvent, pb *pocketbase.PocketBase) {
	// List provider presets (builtin + user-saved files)
	e.Router.GET("/api/aisw/presets", func(ev *core.RequestEvent) error {
		presets, err := templates.AllPresets()
		if err != nil {
			return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusOK, presets)
	})

	// Save a stored provider as a reusable preset file (API keys excluded)
	e.Router.POST("/api/aisw/presets", func(ev *core.RequestEvent) error {
		var payload struct {
			Slug string `json:"slug"`
		}
		if err := json.NewDecoder(ev.Request.Body).Decode(&payload); err != nil || payload.Slug == "" {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "slug is required"})
		}
		s := store.New(pb)
		provider, err := s.GetProvider(payload.Slug)
		if err != nil || provider == nil {
			return ev.JSON(http.StatusNotFound, map[string]any{"ok": false, "message": "provider not found: " + payload.Slug})
		}
		path, err := templates.SaveUserPreset(templates.PresetFromProvider(*provider))
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusCreated, map[string]any{"ok": true, "path": path})
	})

	// Import preset TOML content uploaded from a file
	e.Router.POST("/api/aisw/presets/import", func(ev *core.RequestEvent) error {
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(ev.Request.Body).Decode(&payload); err != nil || strings.TrimSpace(payload.Content) == "" {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "content is required"})
		}
		tmp, err := os.CreateTemp("", "aisw-preset-*.toml")
		if err != nil {
			return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.WriteString(payload.Content); err != nil {
			tmp.Close()
			return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
		}
		tmp.Close()
		imported, err := templates.ImportUserPresetFile(tmp.Name())
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		slugs := make([]string, 0, len(imported))
		for _, preset := range imported {
			slugs = append(slugs, preset.Slug)
		}
		return ev.JSON(http.StatusCreated, map[string]any{"ok": true, "presets": slugs})
	})

	// Delete a user-saved preset (builtin presets are rejected)
	e.Router.DELETE("/api/aisw/presets/{slug}", func(ev *core.RequestEvent) error {
		if err := templates.DeleteUserPreset(ev.Request.PathValue("slug")); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		return ev.JSON(http.StatusOK, map[string]any{"ok": true})
	})

	// Create a vendor provider from a preset — one API key, per-protocol
	// variants and the preset model list. Looks up builtin AND user presets.
	e.Router.POST("/api/aisw/providers/from-preset", func(ev *core.RequestEvent) error {
		var payload struct {
			PresetSlug string `json:"preset_slug"`
			APIKey     string `json:"api_key"`
		}
		if err := json.NewDecoder(ev.Request.Body).Decode(&payload); err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid JSON: " + err.Error()})
		}
		preset, err := templates.FindSourcedPreset(payload.PresetSlug)
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		provider := templates.ProviderFromPreset(preset.ProviderPreset, payload.APIKey)
		s := store.New(pb)
		result, err := s.UpsertProvider(provider)
		if err != nil {
			return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		}
		result.APIKey = maskAPIKey(result.APIKey)
		return ev.JSON(http.StatusCreated, result)
	})
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func providerCommand(getPB PBGetter) *cobra.Command {
	cmd := &cobra.Command{Use: "provider", Short: "Manage shared LLM providers"}

	var add store.Provider
	var endpointFlags []string
	var modelsFlag []string
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
			if len(modelsFlag) > 0 {
				add.Models = append(add.Models, modelsFlag...)
			}
			if add.DefaultModel == "" && len(add.Models) > 0 {
				add.DefaultModel = add.Models[0]
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
	addCmd.Flags().StringVar(&add.APIProtocol, "protocol", "openai_chat", "anthropic|openai_chat|openai_responses")
	addCmd.Flags().StringVar(&add.DefaultModel, "model", "", "default model")
	addCmd.Flags().StringSliceVar(&modelsFlag, "models", nil, "model list (comma-separated); models share the provider API key")
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
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "# Saved providers")
			for _, provider := range providers {
				keyState := "missing"
				if provider.APIKey != "" {
					keyState = "set"
				}
				protocols := provider.APIProtocol
				if len(provider.Variants) > 0 {
					protocols = strings.Join(providerVariantProtocols(provider), ",")
				}
				models := provider.DefaultModel
				if len(provider.Models) > 0 {
					models = strings.Join(provider.Models, ",")
				}
				fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\tkey=%s\n", provider.Slug, provider.Name, protocols, models, provider.BaseURL, keyState)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "# Built-in presets")
			presets, err := templates.ProviderPresets()
			if err != nil {
				return err
			}
			tui.PrintPresets(presets)
			return nil
		},
	}

	fromPresetCmd := fromPresetCommand(getPB)

	modelCmd := providerModelCommand(getPB)

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

	cmd.AddCommand(addCmd, listCmd, fromPresetCmd, presetCommands(getPB), modelCmd, deleteCmd)
	return cmd
}

// fromPresetCommand creates a vendor provider from a bundled preset with a
// single API key — the preset carries the per-protocol endpoints and models.
func fromPresetCommand(getPB PBGetter) *cobra.Command {
	var apiKey string
	var apiKeyEnv string
	var modelsFlag []string
	cmd := &cobra.Command{
		Use:   "from-preset PRESET_SLUG",
		Short: "Add a vendor provider from a preset (builtin or user-saved file)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			preset, err := templates.FindSourcedPreset(args[0])
			if err != nil {
				return err
			}
			if apiKey == "" && apiKeyEnv != "" {
				apiKey = os.Getenv(apiKeyEnv)
			}
			provider := templates.ProviderFromPreset(preset.ProviderPreset, apiKey)
			if len(modelsFlag) > 0 {
				provider.Models = append(provider.Models, modelsFlag...)
			}
			saved, err := store.New(pb).UpsertProvider(provider)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved provider %s (models: %s, preset source: %s)\n", saved.Slug, strings.Join(saved.Models, ", "), preset.Source)
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key shared by every model and agent")
	cmd.Flags().StringVar(&apiKeyEnv, "api-key-env", "", "read API key from an environment variable")
	cmd.Flags().StringSliceVar(&modelsFlag, "models", nil, "extra models to register in addition to the preset list")
	return cmd
}

// presetCommands save providers as reusable preset files and import preset
// files — user presets live in ~/.innate-aiswitcher/presets/*.toml and are
// picked up by provider list / from-preset automatically.
func presetCommands(getPB PBGetter) *cobra.Command {
	cmd := &cobra.Command{Use: "preset", Short: "Manage reusable provider presets (saved as TOML files)"}

	saveCmd := &cobra.Command{
		Use:   "save PROVIDER_SLUG",
		Short: "Save a stored provider as a preset file (API key excluded)",
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
			path, err := templates.SaveUserPreset(templates.PresetFromProvider(*provider))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved preset %s -> %s (API key excluded)\n", provider.Slug, path)
			return nil
		},
	}

	importCmd := &cobra.Command{
		Use:   "import PATH",
		Short: "Import presets from a TOML file into the user presets directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imported, err := templates.ImportUserPresetFile(args[0])
			if err != nil {
				return err
			}
			dir, _ := templates.UserPresetsDir()
			for _, preset := range imported {
				fmt.Fprintf(cmd.OutOrStdout(), "imported preset %s (models: %s)\n", preset.Slug, strings.Join(preset.Models, ", "))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "presets directory: %s\n", dir)
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List presets (builtin + user files) with their source",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			presets, err := templates.AllPresets()
			if err != nil {
				return err
			}
			for _, preset := range presets {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", preset.Slug, preset.Name, strings.Join(preset.Models, ","), preset.Source)
			}
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete SLUG",
		Short: "Delete a user-saved preset file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := templates.DeleteUserPreset(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deleted preset %s\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(saveCmd, importCmd, listCmd, deleteCmd)
	return cmd
}

// providerModelCommand manages a provider's model list. Models share the
// provider's stored API key — no key handling required.
func providerModelCommand(getPB PBGetter) *cobra.Command {
	cmd := &cobra.Command{Use: "model", Short: "Manage a provider's model list (models share the provider API key)"}

	listCmd := &cobra.Command{
		Use:   "list SLUG",
		Short: "List configured models",
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
			out := cmd.OutOrStdout()
			for _, model := range provider.Models {
				marker := ""
				if model == provider.DefaultModel {
					marker = " (default)"
				}
				fmt.Fprintf(out, "%s%s\n", model, marker)
			}
			return nil
		},
	}

	var setDefault bool
	addCmd := &cobra.Command{
		Use:   "add SLUG MODEL",
		Short: "Add a model to a provider; it shares the stored API key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			saved, err := store.New(pb).AddModel(args[0], args[1], setDefault)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "added model %s to %s (models: %s, default: %s)\n",
				args[1], saved.Slug, strings.Join(saved.Models, ", "), saved.DefaultModel)
			return nil
		},
	}
	addCmd.Flags().BoolVar(&setDefault, "default", false, "make this model the provider default")

	removeCmd := &cobra.Command{
		Use:   "remove SLUG MODEL",
		Short: "Remove a model from a provider",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			saved, err := store.New(pb).RemoveModel(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed model %s from %s (models: %s, default: %s)\n",
				args[1], saved.Slug, strings.Join(saved.Models, ", "), saved.DefaultModel)
			return nil
		},
	}

	cmd.AddCommand(listCmd, addCmd, removeCmd)
	return cmd
}

func providerVariantProtocols(provider store.Provider) []string {
	protocols := make([]string, 0, len(provider.Variants))
	for _, protocol := range []string{"anthropic", "openai_responses", "openai_chat"} {
		if _, ok := provider.Variants[protocol]; ok {
			protocols = append(protocols, protocol)
		}
	}
	return protocols
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
			if profile.IsDefault {
				agent, err := store.New(pb).GetAgent(profile.AgentSlug)
				if err != nil || agent == nil {
					return fmt.Errorf("agent not found: %s", profile.AgentSlug)
				}
				provider, err := store.New(pb).GetProvider(profile.ProviderSlug)
				if err != nil || provider == nil {
					return fmt.Errorf("provider not found: %s", profile.ProviderSlug)
				}
				if err := adapter.PersistDefault(*agent, *provider, result); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to persist default config for %s: %v\n", agent.Name, err)
				}
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
	addCmd.Flags().StringVar(&profile.SkipPermissions, "skip-permissions", "", "override agent's skip-permissions default: true | false (empty = use agent default)")
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
	var modelOverride string
	cmd := &cobra.Command{
		Use:   "start AGENT [PROVIDER_OR_PROFILE] [--model M] -- [native args]",
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
			opts := adapter.LaunchOptions{CWD: cwd, Terminal: terminal, DryRun: dryRun, Args: nativeArgs, Model: modelOverride}
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
	cmd.Flags().StringVar(&modelOverride, "model", "", "model override (must be configured on the provider)")
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
			result, err := httpcheck.CheckProvider(context.Background(), nil, providerconfig.ResolveDefault(*provider), model)
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
			result, err := httpcheck.ListModels(context.Background(), nil, providerconfig.ResolveDefault(*provider))
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
	cmd := &cobra.Command{Use: "config", Short: "Import/export the shared config mirror (TOML or JSON)"}
	var path string
	var includeSecrets bool
	var backup bool
	var noBackup bool
	var backupPath string
	var format string
	parseFormat := func(defaultFmt configfile.Format) (configfile.Format, error) {
		f := configfile.Format(format)
		if f == "" {
			return defaultFmt, nil
		}
		if !f.Valid() {
			return "", fmt.Errorf("unsupported --format %q (expected toml or json)", format)
		}
		return f, nil
	}
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export SQLite data to a TOML or JSON config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			if path == "" {
				path = configfile.DefaultPath()
			}
			f, err := parseFormat(configfile.FormatTOML)
			if err != nil {
				return err
			}
			if err := configfile.Export(store.New(pb), path, f, includeSecrets); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %s (%s)\n", path, f)
			return nil
		},
	}
	exportCmd.Flags().StringVar(&path, "path", "", "config path")
	exportCmd.Flags().BoolVar(&includeSecrets, "include-secrets", false, "include API keys in the exported file")
	exportCmd.Flags().StringVar(&format, "format", "", "wire format: toml (default) or json")

	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "Export full config (with secrets) to the init-config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			if path == "" {
				path = configfile.InitConfigPath()
			}
			f, err := parseFormat(configfile.FormatTOML)
			if err != nil {
				return err
			}
			if err := configfile.DumpWithFormat(store.New(pb), path, f); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "dumped %s (%s)\n", path, f)
			return nil
		},
	}
	dumpCmd.Flags().StringVar(&path, "path", "", "dump path")
	dumpCmd.Flags().StringVar(&format, "format", "", "wire format: toml (default) or json")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Backup current SQLite config and import a TOML or JSON config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			if path == "" {
				path = configfile.DefaultPath()
			}
			f, err := parseFormat("")
			if err != nil {
				return err
			}
			s := store.New(pb)
			if backup && !noBackup {
				actualBackupPath, err := configfile.Backup(s, backupPath)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "backup %s\n", actualBackupPath)
			}
			if err := configfile.Import(s, path, f); err != nil {
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
	importCmd.Flags().StringVar(&format, "format", "", "wire format: toml, json, or auto-detect (default)")

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

	cmd.AddCommand(exportCmd, dumpCmd, importCmd, templateCmd)
	return cmd
}

func serveCommand(getPB PBGetter, opts *Options) *cobra.Command {
	var allowedOrigins []string
	var httpAddr string
	var httpsAddr string
	var quiet bool
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
			return runServe(cmd, getPB, opts, allowedOrigins, httpAddr, httpsAddr, args, false, quiet)
		},
	}
	cmd.Flags().BoolVar(&quiet, "quiet", false, "disable HTTP access logs")
	cmd.PersistentFlags().StringSliceVar(&allowedOrigins, "origins", []string{"*"}, "CORS allowed domain origins list")
	cmd.PersistentFlags().StringVar(&httpAddr, "http", "", "TCP address to listen for HTTP")
	cmd.PersistentFlags().StringVar(&httpsAddr, "https", "", "TCP address to listen for HTTPS")
	return cmd
}

// webCommand starts the same server as `serve` and additionally opens the
// web UI in the local browser — the single-binary way to run AISwitcher.
func webCommand(getPB PBGetter, opts *Options) *cobra.Command {
	var httpAddr string
	var httpsAddr string
	var allowedOrigins []string
	var noBrowser bool
	var quiet bool
	cmd := &cobra.Command{
		Use:          "web",
		Short:        "Start the web app (providers, profiles and terminal sessions) and open it locally",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if httpAddr == "" {
				httpAddr = "127.0.0.1:8090"
			}
			return runServe(cmd, getPB, opts, allowedOrigins, httpAddr, httpsAddr, nil, !noBrowser, quiet)
		},
	}
	cmd.Flags().StringVar(&httpAddr, "http", "", "TCP address to listen for HTTP (default 127.0.0.1:8090)")
	cmd.Flags().StringVar(&httpsAddr, "https", "", "TCP address to listen for HTTPS")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open the browser automatically")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "disable HTTP access logs")
	cmd.PersistentFlags().StringSliceVar(&allowedOrigins, "origins", []string{"*"}, "CORS allowed domain origins list")
	return cmd
}

func runServe(cmd *cobra.Command, getPB PBGetter, opts *Options, allowedOrigins []string, httpAddr, httpsAddr string, certificateDomains []string, openBrowser bool, quiet bool) error {
	opts.DisableAccessLog = quiet
	logServeStartup(opts.DataDir, httpAddr)
	warnTerminalExposure(httpAddr)

	if openBrowser {
		go func() {
			time.Sleep(600 * time.Millisecond)
			openBrowserAt("http://" + strings.Split(httpAddr, ":")[0] + ":" + lastPort(httpAddr))
		}()
	}

	pb, err := getPB()
	if err != nil {
		return err
	}
	logServeReady(httpAddr)
	err = apis.Serve(pb, apis.ServeConfig{
		HttpAddr:           httpAddr,
		HttpsAddr:          httpsAddr,
		ShowStartBanner:    true,
		AllowedOrigins:     allowedOrigins,
		CertificateDomains: certificateDomains,
	})
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func lastPort(addr string) string {
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[idx+1:]
	}
	return addr
}

func openBrowserAt(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}

// warnTerminalExposure prints a warning when browser terminal sessions
// (local shells) would be reachable from beyond this machine.
func warnTerminalExposure(httpAddr string) {
	host, _, err := net.SplitHostPort(httpAddr)
	if err != nil {
		return
	}
	ip := net.ParseIP(host)
	if ip != nil && !ip.IsLoopback() {
		fmt.Fprintln(os.Stderr, terminal.LoopbackWarning(httpAddr))
	}
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

func maybeInitConfig(pb *pocketbase.PocketBase, initConfigPath string) error {
	s := store.New(pb)
	providers, err := s.ListProviders()
	if err != nil {
		return err
	}
	profiles, err := s.ListProfiles()
	if err != nil {
		return err
	}
	if len(providers) > 0 || len(profiles) > 0 {
		return nil
	}

	path := initConfigPath
	if path == "" {
		path = configfile.InitConfigPath()
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if initConfigPath != "" {
			return fmt.Errorf("init-config file not found: %s", path)
		}
		return nil
	}

	if err := configfile.Import(s, path, ""); err != nil {
		return fmt.Errorf("init-config import failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "initialized from %s\n", path)
	return nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
