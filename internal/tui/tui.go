package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/variableway/innate-aiswitcher/internal/adapter"
	"github.com/variableway/innate-aiswitcher/internal/httpcheck"
	"github.com/variableway/innate-aiswitcher/internal/store"
	"github.com/variableway/innate-aiswitcher/internal/templates"
)

func Run(s *store.Store) error {
	providers, err := s.ListProviders()
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		fmt.Println("No providers configured. Let's add one from a bundled template.")
		return configureProvider(s)
	}

	action := "start"
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Action").Options(
			huh.NewOption("Start an agent session", "start"),
			huh.NewOption("List providers", "list"),
			huh.NewOption("Configure provider", "configure"),
			huh.NewOption("Test provider", "test"),
		).Value(&action),
	)).Run(); err != nil {
		return err
	}

	switch action {
	case "list":
		printProviders(providers)
		return nil
	case "configure":
		return configureProvider(s)
	case "test":
		return testProvider(providers)
	default:
		return startSession(s, providers)
	}
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
	providerOptions := make([]huh.Option[string], 0, len(providers))
	for _, provider := range providers {
		if provider.Active {
			providerOptions = append(providerOptions, huh.NewOption(providerSelectLabel(provider), provider.Slug))
		}
	}

	var agentSlug string
	var selector string
	var dryRun = true
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Agent").Options(agentOptions...).Value(&agentSlug),
			huh.NewSelect[string]().Title("Provider").Options(providerOptions...).Value(&selector),
			huh.NewConfirm().Title("Dry run only?").Affirmative("yes").Negative("start now").Value(&dryRun),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	agent, provider, profile, err := s.ResolveSelector(agentSlug, selector)
	if err != nil {
		return err
	}
	cwd, _ := os.Getwd()
	plan, cleanup, err := adapter.BuildPlan(*agent, *provider, profile, adapter.LaunchOptions{CWD: cwd, Terminal: "current", DryRun: dryRun})
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
	return adapter.Execute(plan, func() {}, adapter.LaunchOptions{CWD: cwd, Terminal: "current", DryRun: dryRun})
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
	if preset.Slug == "" || len(preset.URLOptions) == 0 {
		return fmt.Errorf("provider preset has no URL options: %s", presetSlug)
	}

	optionSlug := preset.URLOptions[0].Slug
	if len(preset.URLOptions) > 1 {
		optionOptions := make([]huh.Option[string], 0, len(preset.URLOptions))
		for _, option := range preset.URLOptions {
			optionOptions = append(optionOptions, huh.NewOption(templates.OptionLabel(option), option.Slug))
		}
		if err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().Title("URL format").Options(optionOptions...).Value(&optionSlug),
		)).Run(); err != nil {
			return err
		}
	}

	var option templates.URLOption
	for _, candidate := range preset.URLOptions {
		if candidate.Slug == optionSlug {
			option = candidate
			break
		}
	}
	if option.Slug == "" {
		return fmt.Errorf("URL option not found: %s", optionSlug)
	}

	provider := templates.ProviderFromPreset(preset, option, "")
	apiKey := ""
	baseURL := provider.BaseURL
	model := provider.DefaultModel
	slug := provider.Slug
	name := provider.Name

	if err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Provider slug").Value(&slug),
		huh.NewInput().Title("Display name").Value(&name),
		huh.NewInput().Title("API key").Description("Leave empty to keep an existing saved key.").EchoMode(huh.EchoModePassword).Value(&apiKey),
		huh.NewInput().Title("Base URL").Value(&baseURL),
		huh.NewInput().Title("Default model").Value(&model),
	)).Run(); err != nil {
		return err
	}

	provider.Slug = slug
	provider.Name = name
	provider.APIKey = strings.TrimSpace(apiKey)
	provider.BaseURL = strings.TrimSpace(baseURL)
	provider.DefaultModel = strings.TrimSpace(model)
	if provider.DefaultModel == "" {
		return fmt.Errorf("default model is required for provider %s", provider.Slug)
	}
	saved, err := s.UpsertProvider(provider)
	if err != nil {
		return err
	}
	fmt.Printf("Saved provider %s (%s, model=%s)\n", saved.Slug, saved.BaseURL, saved.DefaultModel)

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
	result, err := httpcheck.CheckProvider(context.Background(), nil, provider, model)
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
	model := provider.DefaultModel
	if model == "" {
		model = "model=missing"
	} else {
		model = "model=" + model
	}
	return fmt.Sprintf("%s | %s | %s | %s | %s | %s", provider.Slug, provider.Name, provider.APIProtocol, model, provider.BaseURL, keyState)
}

// PrintPresets renders provider presets with styled TUI output using lipgloss.
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

	optionTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF8C00")).
		MarginTop(1)

	endpointStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#98FB98"))

	// Title
	fmt.Println(titleStyle.Render("📦 Built-in Provider Presets"))
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

		// URL options
		for i, option := range preset.URLOptions {
			if len(preset.URLOptions) > 1 {
				optionTitle := fmt.Sprintf("  option %d: %s", i+1, option.Label)
				presetContent.WriteString(optionTitleStyle.Render(optionTitle))
				presetContent.WriteString("\n")
			}

			provider := templates.ProviderFromPreset(preset, option, "")

			// Protocol
			presetContent.WriteString(
				lipgloss.JoinHorizontal(lipgloss.Top,
					labelStyle.Render("protocol"),
					valueStyle.Render(provider.APIProtocol),
				),
			)
			presetContent.WriteString("\n")

			// Model
			model := provider.DefaultModel
			if model == "" {
				model = "(none)"
			}
			presetContent.WriteString(
				lipgloss.JoinHorizontal(lipgloss.Top,
					labelStyle.Render("model"),
					valueStyle.Render(model),
				),
			)
			presetContent.WriteString("\n")

			// Base URL
			presetContent.WriteString(
				lipgloss.JoinHorizontal(lipgloss.Top,
					labelStyle.Render("base_url"),
					valueStyle.Render(provider.BaseURL),
				),
			)
			presetContent.WriteString("\n")

			// Endpoints
			if len(provider.Endpoints) > 0 {
				var epParts []string
				for k, v := range provider.Endpoints {
					epParts = append(epParts, fmt.Sprintf("%s=%s", k, v))
				}
				presetContent.WriteString(
					lipgloss.JoinHorizontal(lipgloss.Top,
						labelStyle.Render("endpoints"),
						endpointStyle.Render(strings.Join(epParts, ", ")),
					),
				)
				presetContent.WriteString("\n")
			}
		}

		fmt.Println(presetBoxStyle.Render(presetContent.String()))
	}

	// Footer hint
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true)
	fmt.Println(hintStyle.Render(fmt.Sprintf("Total: %d preset(s)", len(presets))))
}
