package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
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

// Accessible forces huh forms into accessible (line-based) mode. Set by the
// --accessible CLI flag; the HuhAccessible / HUH_ACCESSIBLE env vars also work.
var Accessible bool

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func accessibleMode() bool {
	if Accessible {
		return true
	}
	for _, name := range []string{"HuhAccessible", "HUH_ACCESSIBLE"} {
		if value := os.Getenv(name); value == "1" || strings.EqualFold(value, "true") {
			return true
		}
	}
	return false
}

// newForm builds a huh form with the aisw theme and accessibility settings.
func newForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(aiswTheme()).WithAccessible(accessibleMode())
}

// aiswTheme is ThemeCharm re-tinted to the aisw semantic palette in styles.go.
func aiswTheme() *huh.Theme {
	t := huh.ThemeCharm()

	cream := lipgloss.Color("#FFFDF5")

	t.Focused.Title = t.Focused.Title.Foreground(ColorPrimary).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(ColorPrimary).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(ColorMuted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(ColorDanger)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(ColorDanger)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(ColorWarning)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(ColorWarning)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(ColorWarning)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(ColorWarning)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(ColorSuccess)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(cream).Background(ColorPrimary)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(ColorSuccess)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(ColorWarning)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(ColorMuted)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.MultiSelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}

// isTerminal reports whether f is an interactive terminal (no x/term dep).
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func brandCard() string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		MarginBottom(1).
		Render(SlugStyle.Render("⚡ aisw") + ValueStyle.Render(" · AI Provider Switcher"))
}

// contextSummary is the one-line state digest under the main menu. Every
// store read is fault-tolerant: failed reads are simply omitted.
func contextSummary(s *store.Store) string {
	parts := []string{}
	if providers, err := s.ListProviders(); err == nil {
		parts = append(parts, fmt.Sprintf("%d providers", len(providers)))
	}
	if agents, err := s.ListAgents(); err == nil {
		parts = append(parts, fmt.Sprintf("%d agents", len(agents)))
	}
	if profiles, err := s.ListProfiles(); err == nil {
		for _, profile := range profiles {
			if profile.IsDefault {
				parts = append(parts, fmt.Sprintf("default: %s+%s", profile.AgentSlug, profile.ProviderSlug))
				break
			}
		}
	}
	return strings.Join(parts, " · ")
}

func Run(s *store.Store) error {
	// Non-interactive stdout (pipe, CI, redirect): never block on a form.
	if !isTerminal(os.Stdout) {
		fmt.Println(HintStyle.Render("aisw TUI needs an interactive terminal — showing the saved providers instead."))
		providers, err := s.ListProviders()
		if err != nil {
			return err
		}
		PrintProviders(providers)
		return nil
	}

	fmt.Println(brandCard())
	for {
		providers, err := s.ListProviders()
		if err != nil {
			return err
		}
		if len(providers) == 0 {
			fmt.Println(HintStyle.Render("No providers configured. Let's add one from a bundled template."))
			if err := configureProvider(s); err != nil {
				if !errors.Is(err, huh.ErrUserAborted) {
					fmt.Fprintf(os.Stderr, "configure: %v\n", err)
				}
				return nil
			}
			continue
		}

		action := "start"
		if err := newForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Action").
				Description(contextSummary(s)).
				Options(
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
			PrintProviders(providers)
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
			if err := startSession(s, providers); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}
		}
	}
}

// marketPickLimit caps how many catalog entries the multi-select renders;
// the search prompt narrows the list below it.
const marketPickLimit = 100

// importMarketModels pulls models from the local market catalog (models.dev
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
			huh.NewConfirm().Title("Fetch the catalog from models.dev now?").Affirmative("fetch").Negative("cancel").Value(&fetchNow),
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

	agentBySlug := map[string]store.Agent{}
	agentOptions := make([]huh.Option[string], 0, len(agents))
	for _, agent := range agents {
		if agent.Active {
			agentBySlug[agent.Slug] = agent
			agentOptions = append(agentOptions, huh.NewOption(agent.Slug+" · "+agent.Name, agent.Slug))
		}
	}
	if len(agentOptions) == 0 {
		return fmt.Errorf("no active agents configured")
	}

	providerBySlug := map[string]store.Provider{}
	for _, provider := range providers {
		if provider.Active {
			providerBySlug[provider.Slug] = provider
		}
	}
	if len(providerBySlug) == 0 {
		return fmt.Errorf("no active providers configured; add one from a preset first")
	}

	var agentSlug, selector, modelOverride string
	startNow := true

	// Provider options re-filter whenever the chosen agent changes; model
	// options follow the chosen provider. One form, four navigable groups.
	form := newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Agent").
				Options(agentOptions...).
				DescriptionFunc(func() string {
					agent, ok := agentBySlug[agentSlug]
					if !ok {
						return ""
					}
					return "adapter: " + agent.Adapter
				}, &agentSlug).
				Value(&agentSlug),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Provider").
				OptionsFunc(func() []huh.Option[string] {
					options := make([]huh.Option[string], 0, len(providerBySlug))
					for _, provider := range providers {
						if provider.Active && providerconfig.SupportsAgent(provider, agentAdapter(s, agentSlug)) {
							options = append(options, huh.NewOption(providerSelectLabel(provider), provider.Slug))
						}
					}
					return options
				}, &agentSlug).
				DescriptionFunc(func() string {
					return providerDescription(providerBySlug[selector])
				}, &selector).
				Validate(func(value string) error {
					provider, ok := providerBySlug[value]
					if !ok {
						return fmt.Errorf("no active provider supports agent %s; abort and configure one first", agentSlug)
					}
					if !providerconfig.SupportsAgent(provider, agentAdapter(s, agentSlug)) {
						return fmt.Errorf("provider %s does not support agent %s", value, agentSlug)
					}
					return nil
				}).
				Value(&selector),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Model").
				OptionsFunc(func() []huh.Option[string] {
					provider, ok := providerBySlug[selector]
					if !ok {
						return []huh.Option[string]{huh.NewOption("default", "")}
					}
					options := []huh.Option[string]{
						huh.NewOption(fmt.Sprintf("default (%s)", provider.DefaultModel), ""),
					}
					for _, model := range provider.Models {
						if model == provider.DefaultModel {
							continue
						}
						options = append(options, huh.NewOption(model, model))
					}
					return options
				}, &selector).
				Value(&modelOverride),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Start the session now?").
				Description("Dry run prints the launch plan without starting the agent.").
				Affirmative("Start now").
				Negative("Dry run").
				Value(&startNow),
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
	opts := adapter.LaunchOptions{CWD: cwd, Terminal: "current", DryRun: !startNow, Model: modelOverride}
	plan, cleanup, err := adapter.BuildPlan(*agent, *provider, profile, opts)
	if err != nil {
		return err
	}
	defer cleanup()
	fmt.Println(RenderLaunchPlan(agent.Slug, provider.Slug, modelOverride, plan))
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
	if err := newForm(huh.NewGroup(
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
			keyDescription = fmt.Sprintf("Saved key: %s — leave empty to keep it, or enter a new one.", MaskAPIKey(existing.APIKey))
		}
	}

	if err := newForm(huh.NewGroup(
		huh.NewInput().Title("Provider slug").Placeholder("my-provider").
			Validate(func(value string) error {
				if !slugPattern.MatchString(strings.TrimSpace(value)) {
					return fmt.Errorf("slug must match ^[a-z0-9-]+$ (lowercase letters, digits, dashes)")
				}
				return nil
			}).Value(&slug),
		huh.NewInput().Title("Display name").Placeholder(provider.Name).
			Validate(func(value string) error {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("name is required")
				}
				return nil
			}).Value(&name),
		huh.NewInput().Title("API key").Description(keyDescription).Placeholder("sk-...").
			EchoMode(huh.EchoModePassword).Value(&apiKey),
		huh.NewInput().Title("Models").Placeholder("model-a, model-b").
			Description("Comma-separated; all models share the API key.").
			Validate(func(value string) error {
				if len(splitModels(value)) == 0 {
					return fmt.Errorf("at least one model is required")
				}
				return nil
			}).Value(&models),
		huh.NewInput().Title("Default model").Placeholder(provider.DefaultModel).
			Description("Leave empty to use the first model in the list.").Value(&model),
	)).Run(); err != nil {
		return err
	}

	slug = strings.TrimSpace(slug)

	// Renaming the slug would silently orphan the existing provider — confirm.
	if existing != nil && slug != existing.Slug {
		var createNew bool
		if err := newForm(huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Slug changed: %s → %s", existing.Slug, slug)).
				Description("Saving now creates a NEW provider; the existing one stays untouched.").
				Affirmative("Create new").
				Negative("Cancel").
				Value(&createNew),
		)).Run(); err != nil {
			return err
		}
		if !createNew {
			fmt.Println(HintStyle.Render("Cancelled — existing provider unchanged."))
			return nil
		}
	}

	provider.Slug = slug
	provider.Name = strings.TrimSpace(name)
	provider.APIKey = strings.TrimSpace(apiKey)
	if provider.APIKey == "" && existing != nil && slug == existing.Slug {
		// Empty key on an unchanged slug keeps the stored key.
		provider.APIKey = existing.APIKey
	}
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
	fmt.Println(RenderSavedProvider(*saved))

	if saved.APIKey != "" {
		var testNow bool
		if err := newForm(huh.NewGroup(
			huh.NewConfirm().Title("Test this provider now?").Affirmative("Test").Negative("Skip").Value(&testNow),
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
	if err := newForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Provider").
			Options(providerOptions...).
			DescriptionFunc(func() string {
				return providerDescription(providerBySlug[providerSlug])
			}, &providerSlug).
			Value(&providerSlug),
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

// providerSelectLabel is the compact select-option label: slug · name only.
// Details (protocols/models/key state) live in the field description.
func providerSelectLabel(provider store.Provider) string {
	return provider.Slug + " · " + provider.Name
}

// providerDescription renders the second information layer for provider
// selects. Key status matches the cards: ✓ key set / ✗ key missing.
func providerDescription(provider store.Provider) string {
	if provider.Slug == "" {
		return ""
	}
	protocols := strings.Join(VariantProtocols(provider), ", ")
	models := strings.Join(provider.Models, ", ")
	if models == "" {
		models = "(any)"
	}
	keyState := "✗ key missing"
	if provider.APIKey != "" {
		keyState = "✓ key set"
	}
	return fmt.Sprintf("protocols: %s · models: %s · %s", protocols, models, keyState)
}

// PrintPresets renders builtin vendor provider presets as styled cards.
func PrintPresets(presets []templates.ProviderPreset) {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("📦 Built-in Provider Presets (one API key per vendor)"))
	b.WriteString("\n")
	for _, preset := range presets {
		b.WriteString(RenderPresetCard(preset, ""))
		b.WriteString("\n")
	}
	b.WriteString(HintStyle.Render(fmt.Sprintf("Total: %d preset(s) — add with: aisw provider from-preset <slug> --api-key <key>", len(presets))))
	fmt.Println(b.String())
}
