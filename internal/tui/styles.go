// Shared semantic colors, lipgloss styles and card renderers for every
// user-facing terminal surface (TUI and CLI). Keep all hard-coded hex
// values here so the whole CLI speaks one visual language.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/variableway/innate-aiswitcher/internal/adapter"
	"github.com/variableway/innate-aiswitcher/internal/store"
	"github.com/variableway/innate-aiswitcher/internal/templates"
)

// Semantic palette.
var (
	ColorPrimary  = lipgloss.Color("#87CEEB") // sky blue — labels, brand border
	ColorSuccess  = lipgloss.Color("#7CFC00") // titles, ✓ states
	ColorDanger   = lipgloss.Color("#FF6B6B") // ✗ states, errors
	ColorMuted    = lipgloss.Color("#808080") // hints
	ColorWarning  = lipgloss.Color("#FFD700") // gold — slugs
	ColorValue    = lipgloss.Color("#E0E0E0") // values
	ColorBorder   = lipgloss.Color("#3C3C3C") // neutral card border
	ColorProtocol = lipgloss.Color("#98FB98") // protocol names / URLs
)

// Shared styles.
var (
	TitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).MarginBottom(1)
	CardStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorBorder).Padding(0, 1).MarginBottom(1)
	SuccessCardStyle = CardStyle.BorderForeground(ColorSuccess)
	LabelStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Width(LabelWidth)
	ValueStyle       = lipgloss.NewStyle().Foreground(ColorValue)
	SlugStyle        = lipgloss.NewStyle().Bold(true).Foreground(ColorWarning)
	ProtocolStyle    = lipgloss.NewStyle().Foreground(ColorProtocol)
	HintStyle        = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)
	SuccessStyle     = lipgloss.NewStyle().Foreground(ColorSuccess)
	DangerStyle      = lipgloss.NewStyle().Foreground(ColorDanger)
	WarningStyle     = lipgloss.NewStyle().Bold(true).Foreground(ColorWarning)
)

// LabelWidth is the aligned label column used by every card. It must fit the
// longest label in use: "openai_responses" (16 chars).
const LabelWidth = 16

// MaskAPIKey redacts a secret for terminal output, keeping short prefixes and
// suffixes so users can still recognize which key is configured.
func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// KeyState renders the canonical key status line used by the TUI selects,
// the CLI provider list and the save confirmation card.
func KeyState(apiKey string) string {
	if apiKey != "" {
		return SuccessStyle.Render("✓ key set")
	}
	return DangerStyle.Render("✗ key missing")
}

// VariantProtocols lists the wire protocols a provider serves, in canonical
// order, falling back to the legacy top-level api_protocol.
func VariantProtocols(provider store.Provider) []string {
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

// labelValue renders one aligned `label value` card line.
func labelValue(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, LabelStyle.Render(label), ValueStyle.Render(value))
}

// ProviderCard renders one saved provider as a rounded-border card. Shared by
// the CLI (`aisw provider list`) and the TUI main-menu list.
func ProviderCard(provider store.Provider) string {
	var b strings.Builder
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		SlugStyle.Render(provider.Slug),
		lipgloss.NewStyle().Width(2).Render(" "),
		ValueStyle.Render(provider.Name),
	))
	b.WriteString("\n")
	b.WriteString(labelValue("protocols", ProtocolStyle.Render(strings.Join(VariantProtocols(provider), ", "))))
	b.WriteString("\n")
	models := strings.Join(provider.Models, ", ")
	if models == "" {
		models = "(any model)"
	}
	b.WriteString(labelValue("models", models))
	b.WriteString("\n")
	def := provider.DefaultModel
	if def == "" {
		def = "(none)"
	}
	b.WriteString(labelValue("default model", def))
	b.WriteString("\n")
	b.WriteString(labelValue("key", KeyState(provider.APIKey)))
	return CardStyle.Render(b.String())
}

// RenderProviders renders the full "Saved Providers" section.
func RenderProviders(providers []store.Provider) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("🔌 Saved Providers"))
	b.WriteString("\n")
	if len(providers) == 0 {
		b.WriteString(HintStyle.Render("No providers yet — add one with: aisw provider from-preset <slug> --api-key <key>"))
		b.WriteString("\n")
		return b.String()
	}
	for _, provider := range providers {
		b.WriteString(ProviderCard(provider))
		b.WriteString("\n")
	}
	b.WriteString(HintStyle.Render(fmt.Sprintf("Total: %d provider(s) — add more with: aisw provider from-preset <slug> --api-key <key>", len(providers))))
	b.WriteString("\n")
	return b.String()
}

// PrintProviders prints the saved-provider cards to stdout.
func PrintProviders(providers []store.Provider) {
	fmt.Println(RenderProviders(providers))
}

// RenderSavedProvider renders the green confirmation card after a provider is
// saved from the TUI configure flow.
func RenderSavedProvider(provider store.Provider) string {
	var b strings.Builder
	b.WriteString(SuccessStyle.Render("✓ Saved provider"))
	b.WriteString("\n")
	b.WriteString(labelValue("slug", SlugStyle.Render(provider.Slug)))
	b.WriteString("\n")
	b.WriteString(labelValue("name", provider.Name))
	b.WriteString("\n")
	b.WriteString(labelValue("models", strings.Join(provider.Models, ", ")))
	b.WriteString("\n")
	b.WriteString(labelValue("protocols", ProtocolStyle.Render(strings.Join(VariantProtocols(provider), ", "))))
	b.WriteString("\n")
	b.WriteString(labelValue("key", KeyState(provider.APIKey)))
	return SuccessCardStyle.Render(b.String())
}

// RenderProfiles renders the profile list; the default profile gets a ★.
func RenderProfiles(profiles []store.Profile) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("👤 Profiles"))
	b.WriteString("\n")
	if len(profiles) == 0 {
		b.WriteString(HintStyle.Render("No profiles yet — add one with: aisw profile add <slug> --agent <a> --provider <p>"))
		b.WriteString("\n")
		return b.String()
	}
	for _, profile := range profiles {
		var card strings.Builder
		header := SlugStyle.Render(profile.Slug)
		if profile.IsDefault {
			header = WarningStyle.Render("★ ") + header
		}
		card.WriteString(header)
		card.WriteString("\n")
		card.WriteString(labelValue("agent", profile.AgentSlug))
		card.WriteString("\n")
		card.WriteString(labelValue("provider", profile.ProviderSlug))
		card.WriteString("\n")
		model := profile.Model
		if model == "" {
			model = "(provider default)"
		}
		card.WriteString(labelValue("model", model))
		b.WriteString(CardStyle.Render(card.String()))
		b.WriteString("\n")
	}
	b.WriteString(HintStyle.Render(fmt.Sprintf("Total: %d profile(s)", len(profiles))))
	b.WriteString("\n")
	return b.String()
}

// PrintProfiles prints the profile cards to stdout.
func PrintProfiles(profiles []store.Profile) {
	fmt.Println(RenderProfiles(profiles))
}

// RenderModels renders a provider's model list; the default model is marked
// with a green ●.
func RenderModels(provider store.Provider) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render(fmt.Sprintf("🧠 Models · %s", provider.Slug)))
	b.WriteString("\n")
	if len(provider.Models) == 0 {
		b.WriteString(HintStyle.Render("No configured models — the provider accepts any model name."))
		b.WriteString("\n")
		return b.String()
	}
	var card strings.Builder
	for i, model := range provider.Models {
		if i > 0 {
			card.WriteString("\n")
		}
		if model == provider.DefaultModel {
			card.WriteString(SuccessStyle.Render("● " + model + " (default)"))
		} else {
			card.WriteString(ValueStyle.Render("○ " + model))
		}
	}
	b.WriteString(CardStyle.Render(card.String()))
	b.WriteString("\n")
	b.WriteString(HintStyle.Render(fmt.Sprintf("Total: %d model(s) — default: %s", len(provider.Models), provider.DefaultModel)))
	b.WriteString("\n")
	return b.String()
}

// PrintModels prints the model list card to stdout.
func PrintModels(provider store.Provider) {
	fmt.Println(RenderModels(provider))
}

// RenderPresetCard renders one provider preset card. source is "builtin",
// a file path or empty (no source line rendered).
func RenderPresetCard(preset templates.ProviderPreset, source string) string {
	var b strings.Builder
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		SlugStyle.Render(preset.Slug),
		lipgloss.NewStyle().Width(2).Render(" "),
		ValueStyle.Render(preset.Name),
	))
	b.WriteString("\n")
	b.WriteString(labelValue("models", strings.Join(preset.Models, ", ")))
	b.WriteString("\n")
	for _, protocol := range templates.PresetProtocols(preset) {
		for _, variant := range preset.Variants {
			if variant.Protocol != protocol {
				continue
			}
			b.WriteString(labelValue(protocol, ProtocolStyle.Render(variant.BaseURL)))
			b.WriteString("\n")
		}
	}
	if source != "" {
		b.WriteString(labelValue("source", source))
	}
	return CardStyle.Render(strings.TrimRight(b.String(), "\n"))
}

// RenderSourcedPresets renders builtin + user presets with their source.
func RenderSourcedPresets(presets []templates.SourcedPreset) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("📦 Provider Presets (builtin + user files)"))
	b.WriteString("\n")
	for _, preset := range presets {
		source := preset.Source
		if source == "" {
			source = "builtin"
		}
		b.WriteString(RenderPresetCard(preset.ProviderPreset, source))
		b.WriteString("\n")
	}
	b.WriteString(HintStyle.Render(fmt.Sprintf("Total: %d preset(s) — add with: aisw provider from-preset <slug> --api-key <key>", len(presets))))
	b.WriteString("\n")
	return b.String()
}

// PrintSourcedPresets prints sourced preset cards to stdout.
func PrintSourcedPresets(presets []templates.SourcedPreset) {
	fmt.Println(RenderSourcedPresets(presets))
}

// sensitiveEnvName reports whether an env var name looks like a secret.
func sensitiveEnvName(name string) bool {
	upper := strings.ToUpper(name)
	return strings.Contains(upper, "KEY") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET")
}

// RenderEnvLines renders a sorted KEY=value block; sensitive values (names
// containing KEY/TOKEN/SECRET) are masked.
func RenderEnvLines(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, key := range keys {
		if i > 0 {
			b.WriteString("\n")
		}
		value := env[key]
		if sensitiveEnvName(key) {
			value = MaskAPIKey(value)
		}
		b.WriteString(key + "=" + value)
	}
	return b.String()
}

// RenderLaunchPlan renders the dry-run launch plan as a rounded card instead
// of a raw JSON dump. Env values with sensitive names are masked.
func RenderLaunchPlan(agentSlug, providerSlug, model string, plan adapter.LaunchPlan) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("🚀 Launch Plan"))
	b.WriteString("\n")

	var card strings.Builder
	card.WriteString(labelValue("agent", SlugStyle.Render(agentSlug)))
	card.WriteString("\n")
	card.WriteString(labelValue("provider", SlugStyle.Render(providerSlug)))
	card.WriteString("\n")
	if model == "" {
		model = "(provider default)"
	}
	card.WriteString(labelValue("model", model))
	card.WriteString("\n")
	// plan.Command is already the full command line; plan.Args duplicates it
	// in tokenized form, so only Command is rendered.
	card.WriteString(labelValue("command", plan.Command))
	card.WriteString("\n")
	card.WriteString(labelValue("cwd", plan.CWD))
	if env := RenderEnvLines(plan.Env); env != "" {
		card.WriteString("\n")
		card.WriteString(labelValue("env", env))
	}
	if len(plan.Files) > 0 {
		names := make([]string, 0, len(plan.Files))
		for name := range plan.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		lines := make([]string, 0, len(names))
		for _, name := range names {
			lines = append(lines, name+": "+plan.Files[name])
		}
		card.WriteString("\n")
		card.WriteString(labelValue("temp files", strings.Join(lines, "\n"+strings.Repeat(" ", LabelWidth))))
	}
	b.WriteString(CardStyle.Render(card.String()))
	b.WriteString("\n")
	return b.String()
}
