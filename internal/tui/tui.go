package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/variableway/innate-aiswitcher/internal/adapter"
	"github.com/variableway/innate-aiswitcher/internal/httpcheck"
	"github.com/variableway/innate-aiswitcher/internal/market"
	"github.com/variableway/innate-aiswitcher/internal/providerconfig"
	"github.com/variableway/innate-aiswitcher/internal/store"
	"github.com/variableway/innate-aiswitcher/internal/templates"
)

func Run(s *store.Store) error {
	for {
		providers, err := s.ListProviders()
		if err != nil {
			return err
		}
		if len(providers) == 0 {
			fmt.Println("No providers configured. Let's add one from a bundled template.")
			if err := configureProvider(s); err != nil {
				if !errors.Is(err, huh.ErrUserAborted) {
					fmt.Fprintf(os.Stderr, "configure: %v\n", err)
				}
				return nil
			}
			continue
		}

		action := "start"
		if err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().Title("Action").Options(
				huh.NewOption("Start an agent session", "start"),
				huh.NewOption("List providers", "list"),
				huh.NewOption("Configure provider", "configure"),
				huh.NewOption("Import models from market", "market"),
				huh.NewOption("Test provider", "test"),
				huh.NewOption("Quit", "quit"),
			).Value(&action),
		)).Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		switch action {
		case "quit":
			return nil
		case "list":
			printProviders(providers)
		case "configure":
			if err := configureProvider(s); err != nil && !errors.Is(err, huh.ErrUserAborted) {
				fmt.Fprintf(os.Stderr, "configure: %v\n", err)
			}
		case "market":
			if err := importMarketModels(s, providers); err != nil && !errors.Is(err, huh.ErrUserAborted) {
				fmt.Fprintf(os.Stderr, "market import: %v\n", err)
			}
		case "test":
			if err := testProvider(providers); err != nil && !errors.Is(err, huh.ErrUserAborted) {
				fmt.Fprintf(os.Stderr, "test: %v\n", err)
			}
		default:
			return startSession(s, providers)
		}
	}
}

// marketPickLimit caps how many catalog entries the multi-select renders;
// the search prompt narrows the list below it.
const marketPickLimit = 100

// importMarketModels pulls models from the local market catalog (lobehub
// snapshot fetched via the Web UI or fetched on demand) into a vendor
// provider. Imported models join the provider's model list and share its
// single API key, exactly like `aisw provider model add`.
func importMarketModels(s *store.Store, providers []store.Provider) error {
	catalog, err := market.LoadCatalog(s, market.Filter{})
	if err != nil {
		return err
	}
	if !catalog.HasData {
		fmt.Println("No local market catalog data yet.")
		var fetchNow bool
		if err := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title("Fetch the catalog from lobehub now?").Affirmative("fetch").Negative("cancel").Value(&fetchNow),
		)).Run(); err != nil {
			return err
		}
		if !fetchNow {
			return nil
		}
		result, err := market.FetchAndStore(context.Background(), s, market.LoadSettings(s), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Fetched %d models across %d vendor categories.\n", result.Fetched, result.Categories)
		catalog, err = market.LoadCatalog(s, market.Filter{})
		if err != nil {
			return err
		}
	}

	providerOptions := make([]huh.Option[string], 0, len(providers))
	providerBySlug := map[string]store.Provider{}
	for _, provider := range providers {
		if provider.Active {
			providerOptions = append(providerOptions, huh.NewOption(providerSelectLabel(provider), provider.Slug))
			providerBySlug[provider.Slug] = provider
		}
	}
	if len(providerOptions) == 0 {
		return fmt.Errorf("no active providers configured; add one from a preset first")
	}
	var providerSlug string
	var query string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Provider (models will share its API key)").Options(providerOptions...).Value(&providerSlug),
		huh.NewInput().Title("Search catalog").Description("Optional substring over model id/name; empty lists the first entries.").Value(&query),
	)).Run(); err != nil {
		return err
	}

	if filtered, err := market.LoadCatalog(s, market.Filter{Query: strings.TrimSpace(query)}); err == nil {
		catalog = filtered
	}
	if len(catalog.Items) == 0 {
		return fmt.Errorf("no catalog models match %q", query)
	}

	provider := providerBySlug[providerSlug]
	modelOptions := make([]huh.Option[string], 0, marketPickLimit)
	for i, item := range catalog.Items {
		if i >= marketPickLimit {
			break
		}
		label := fmt.Sprintf("%s (%s)", item.Identifier, item.Category)
		if provider.HasModel(item.Identifier) {
			label += " — already configured"
		}
		modelOptions = append(modelOptions, huh.NewOption(label, item.Identifier))
	}

	var picked []string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().Title("Models").Description("Type to filter; space toggles, enter confirms.").
			Filterable(true).Options(modelOptions...).Value(&picked),
	)).Run(); err != nil {
		return err
	}
	if len(picked) == 0 {
		return nil
	}
	if len(catalog.Items) > marketPickLimit {
		fmt.Printf("Listing capped at %d of %d matches — narrow the search to reach the rest.\n", marketPickLimit, len(catalog.Items))
	}

	saved, err := s.AddModels(providerSlug, picked)
	if err != nil {
		return err
	}
	fmt.Printf("Imported %d model(s) into %s — all models share its API key.\nmodels: %s\n",
		len(picked), saved.Slug, strings.Join(saved.Models, ", "))
	return nil
}

func startSession(s *store.Store, providers []store.Provider) error {
	agents, err := s.ListAgents()
	if err != nil {
		return err
	}

	agentOptions := make([]huh.Option[string], 0, len(agents))
	for _, agent := range agents {
		if agent.Active {
			agentOptions = append(agentOptions, huh.NewOption(agent.Name+" ("+agent.Slug+")", agent.Slug))
		}
	}
	var agentSlug string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Agent").Options(agentOptions...).Value(&agentSlug),
	)).Run(); err != nil {
		return err
	}

	// Only offer providers that can serve the chosen agent.
	providerOptions := make([]huh.Option[string], 0, len(providers))
	for _, provider := range providers {
		if provider.Active && providerconfig.SupportsAgent(provider, agentAdapter(s, agentSlug)) {
			providerOptions = append(providerOptions, huh.NewOption(providerSelectLabel(provider), provider.Slug))
		}
	}
	if len(providerOptions) == 0 {
		return fmt.Errorf("no active providers support agent %s; add one from a preset first", agentSlug)
	}
	var selector string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Provider").Options(providerOptions...).Value(&selector),
	)).Run(); err != nil {
		return err
	}

	agent, provider, profile, err := s.ResolveSelector(agentSlug, selector)
	if err != nil {
		return err
	}

	// Model choice: default plus every configured model (all share the key).
	modelOverride := ""
	if len(provider.Models) > 1 {
		modelOptions := []huh.Option[string]{
			huh.NewOption(fmt.Sprintf("default (%s)", provider.DefaultModel), ""),
		}
		for _, model := range provider.Models {
			if model == provider.DefaultModel {
				continue
			}
			modelOptions = append(modelOptions, huh.NewOption(model, model))
		}
		if err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().Title("Model").Options(modelOptions...).Value(&modelOverride),
		)).Run(); err != nil {
			return err
		}
	}

	dryRun := true
	if err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Dry run only?").Affirmative("yes").Negative("start now").Value(&dryRun),
	)).Run(); err != nil {
		return err
	}

	cwd, _ := os.Getwd()
	opts := adapter.LaunchOptions{CWD: cwd, Terminal: "current", DryRun: dryRun, Model: modelOverride}
	plan, cleanup, err := adapter.BuildPlan(*agent, *provider, profile, opts)
	if err != nil {
		return err
	}
	defer cleanup()
	fmt.Printf("Command: %s\n", plan.Command)
	if len(plan.Env) > 0 {
		fmt.Printf("Env: %v\n", plan.Env)
	}
	if len(plan.Files) > 0 {
		fmt.Printf("Temp files: %v\n", plan.Files)
	}
	return adapter.Execute(plan, func() {}, opts)
}

// agentAdapter maps an agent slug to its adapter name (claude/codex use
// dedicated adapters; everything else speaks the openai env protocol).
func agentAdapter(s *store.Store, slug string) string {
	if agent, err := s.GetAgent(slug); err == nil && agent != nil {
		return agent.Adapter
	}
	return "openai_env"
}

func configureProvider(s *store.Store) error {
	presets, err := templates.ProviderPresets()
	if err != nil {
		return err
	}
	if len(presets) == 0 {
		return fmt.Errorf("no bundled provider presets available")
	}

	presetOptions := make([]huh.Option[string], 0, len(presets))
	for _, preset := range presets {
		presetOptions = append(presetOptions, huh.NewOption(templates.PresetLabel(preset), preset.Slug))
	}

	var presetSlug string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Provider template").Options(presetOptions...).Value(&presetSlug),
	)).Run(); err != nil {
		return err
	}

	var preset templates.ProviderPreset
	for _, candidate := range presets {
		if candidate.Slug == presetSlug {
			preset = candidate
			break
		}
	}
	if preset.Slug == "" {
		return fmt.Errorf("provider preset not found: %s", presetSlug)
	}

	provider := templates.ProviderFromPreset(preset, "")

	// Check if provider already exists in DB — prefill saved values
	existing, _ := s.GetProvider(provider.Slug)
	apiKey := ""
	model := provider.DefaultModel
	models := strings.Join(provider.Models, ", ")
	slug := provider.Slug
	name := provider.Name
	keyDescription := "One key serves claude code, codex and opencode."

	if existing != nil {
		model = existing.DefaultModel
		models = strings.Join(existing.Models, ", ")
		name = existing.Name
		if existing.APIKey != "" {
			keyDescription = "Key already saved. Leave empty to keep it, or enter a new one."
		}
	}

	if err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Provider slug").Value(&slug),
		huh.NewInput().Title("Display name").Value(&name),
		huh.NewInput().Title("API key").Description(keyDescription).EchoMode(huh.EchoModePassword).Value(&apiKey),
		huh.NewInput().Title("Default model").Value(&model),
		huh.NewInput().Title("Models").Description("Comma-separated; all models share the API key.").Value(&models),
	)).Run(); err != nil {
		return err
	}

	provider.Slug = slug
	provider.Name = name
	provider.APIKey = strings.TrimSpace(apiKey)
	provider.DefaultModel = strings.TrimSpace(model)
	provider.Models = splitModels(models)
	if provider.DefaultModel == "" && len(provider.Models) > 0 {
		provider.DefaultModel = provider.Models[0]
	}
	if provider.DefaultModel == "" {
		return fmt.Errorf("default model is required for provider %s", provider.Slug)
	}
	saved, err := s.UpsertProvider(provider)
	if err != nil {
		return err
	}
	fmt.Printf("Saved provider %s (models: %s, endpoints: %s)\n",
		saved.Slug, strings.Join(saved.Models, ", "), strings.Join(variantProtocols(*saved), ", "))

	if saved.APIKey != "" {
		var testNow bool
		if err := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title("Test this provider now?").Affirmative("test").Negative("skip").Value(&testNow),
		)).Run(); err != nil {
			return err
		}
		if testNow {
			return runProviderTest(*saved, "")
		}
	}
	return nil
}

func splitModels(value string) []string {
	parts := strings.Split(value, ",")
	models := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		models = append(models, part)
	}
	return models
}

func variantProtocols(provider store.Provider) []string {
	protocols := make([]string, 0, len(provider.Variants))
	for _, protocol := range []string{"anthropic", "openai_responses", "openai_chat"} {
		if _, ok := provider.Variants[protocol]; ok {
			protocols = append(protocols, protocol)
		}
	}
	if len(protocols) == 0 && provider.APIProtocol != "" {
		protocols = append(protocols, provider.APIProtocol)
	}
	return protocols
}

func testProvider(providers []store.Provider) error {
	providerOptions := make([]huh.Option[string], 0, len(providers))
	providerBySlug := map[string]store.Provider{}
	for _, provider := range providers {
		if provider.Active {
			providerOptions = append(providerOptions, huh.NewOption(providerSelectLabel(provider), provider.Slug))
			providerBySlug[provider.Slug] = provider
		}
	}
	if len(providerOptions) == 0 {
		return fmt.Errorf("no active providers configured")
	}

	var providerSlug string
	var model string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Provider").Options(providerOptions...).Value(&providerSlug),
		huh.NewInput().Title("Model override").Description("Leave empty to use provider default model.").Value(&model),
	)).Run(); err != nil {
		return err
	}
	provider := providerBySlug[providerSlug]
	return runProviderTest(provider, strings.TrimSpace(model))
}

func runProviderTest(provider store.Provider, model string) error {
	result, err := httpcheck.CheckProvider(context.Background(), nil, providerconfig.ResolveDefault(provider), model)
	fmt.Println(httpcheck.Format(result))
	if err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("provider test failed with status %d", result.StatusCode)
	}
	return nil
}

func printProviders(providers []store.Provider) {
	for _, provider := range providers {
		fmt.Println(providerSelectLabel(provider))
	}
}

func providerSelectLabel(provider store.Provider) string {
	keyState := "key=missing"
	if provider.APIKey != "" {
		keyState = "key=set"
	}
	protocols := strings.Join(variantProtocols(provider), ",")
	model := provider.DefaultModel
	if len(provider.Models) > 0 {
		model = strings.Join(provider.Models, ",")
	}
	if model == "" {
		model = "model=missing"
	}
	return fmt.Sprintf("%s | %s | %s | %s | key=%s", provider.Slug, provider.Name, protocols, model, keyState)
}

// PrintPresets renders vendor provider presets with styled TUI output using lipgloss.
func PrintPresets(presets []templates.ProviderPreset) {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7CFC00")).
		MarginBottom(1)

	presetBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C")).
		Padding(0, 1).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87CEEB")).
		Width(14)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))

	slugStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFD700"))

	variantStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#98FB98"))

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true)

	// Title
	fmt.Println(titleStyle.Render("📦 Built-in Provider Presets (one API key per vendor)"))
	fmt.Println()

	for _, preset := range presets {
		var presetContent strings.Builder

		// Preset header: slug + name
		presetContent.WriteString(
			lipgloss.JoinHorizontal(lipgloss.Top,
				slugStyle.Render(preset.Slug),
				lipgloss.NewStyle().Width(2).Render(" "),
				valueStyle.Render(preset.Name),
			),
		)
		presetContent.WriteString("\n")

		// Models
		presetContent.WriteString(
			lipgloss.JoinHorizontal(lipgloss.Top,
				labelStyle.Render("models"),
				valueStyle.Render(strings.Join(preset.Models, ", ")),
			),
		)
		presetContent.WriteString("\n")

		// Per-protocol variants
		for _, protocol := range templates.PresetProtocols(preset) {
			for _, variant := range preset.Variants {
				if variant.Protocol != protocol {
					continue
				}
				presetContent.WriteString(
					lipgloss.JoinHorizontal(lipgloss.Top,
						labelStyle.Render(protocol),
						variantStyle.Render(variant.BaseURL),
					),
				)
				presetContent.WriteString("\n")
			}
		}

		fmt.Println(presetBoxStyle.Render(presetContent.String()))
	}

	fmt.Println(hintStyle.Render(fmt.Sprintf("Total: %d preset(s) — add with: aisw provider from-preset <slug> --api-key <key>", len(presets))))
}
