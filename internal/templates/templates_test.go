package templates

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

var _ = Describe("vendor provider presets", func() {
	Describe("bundled catalog", func() {
		It("contains the bundled vendor catalog", func() {
			presets, err := ProviderPresets()
			Expect(err).NotTo(HaveOccurred())
			slugs := make([]string, 0, len(presets))
			for _, preset := range presets {
				slugs = append(slugs, preset.Slug)
			}
			Expect(slugs).To(ConsistOf("glm", "minimax", "kimi", "deepseek", "openai", "anthropic", "xiaomi"))
		})

		It("finds each vendor by slug", func() {
			for _, slug := range []string{"glm", "minimax", "kimi", "deepseek", "openai", "anthropic", "xiaomi"} {
				preset, err := FindPreset(slug)
				Expect(err).NotTo(HaveOccurred())
				Expect(preset.Slug).To(Equal(slug))
			}
		})

		It("rejects unknown vendors", func() {
			_, err := FindPreset("not-a-vendor")
			Expect(err).To(MatchError(ContainSubstring("not found")))
		})
	})

	Describe("GLM preset", func() {
		It("serves claude (anthropic) and codex/opencode (openai_chat)", func() {
			preset, err := FindPreset("glm")
			Expect(err).NotTo(HaveOccurred())
			Expect(PresetProtocols(*preset)).To(Equal([]string{"anthropic", "openai_chat"}))
			Expect(preset.DefaultModel).To(Equal("glm-5.2"))
		})

		It("ships glm-5.2 and glm-5.3 sharing one key", func() {
			preset, err := FindPreset("glm")
			Expect(err).NotTo(HaveOccurred())
			Expect(preset.Models).To(ConsistOf("glm-5.2", "glm-5.3"))
		})
	})

	Describe("MiniMax preset", func() {
		It("serves every focused agent over three protocols", func() {
			preset, err := FindPreset("minimax")
			Expect(err).NotTo(HaveOccurred())
			Expect(PresetProtocols(*preset)).To(Equal([]string{"anthropic", "openai_responses", "openai_chat"}))

			provider := ProviderFromPreset(*preset, "sk-once")
			Expect(provider.Variants).To(HaveLen(3))
			Expect(provider.Variants["anthropic"].BaseURL).To(Equal("https://api.minimaxi.com/anthropic"))
			Expect(provider.Variants["anthropic"].Endpoints).To(HaveKeyWithValue("messages", "/v1/messages"))
			Expect(provider.Variants["openai_responses"].Capabilities).To(HaveKeyWithValue("codex_auth_mode", "experimental_bearer_token"))
			Expect(provider.Variants["openai_chat"].Endpoints).To(HaveKeyWithValue("chat_completions", "/chat/completions"))
		})
	})

	Describe("projecting a preset into a vendor provider", func() {
		It("carries one API key and the model list on a single row", func() {
			preset, err := FindPreset("glm")
			Expect(err).NotTo(HaveOccurred())

			provider := ProviderFromPreset(*preset, "ark-key")
			Expect(provider.Slug).To(Equal("glm"))
			Expect(provider.Name).To(Equal("GLM (Volcengine Ark)"))
			Expect(provider.APIKey).To(Equal("ark-key"))
			Expect(provider.DefaultModel).To(Equal("glm-5.2"))
			Expect(provider.Models).To(ConsistOf("glm-5.2", "glm-5.3"))
			Expect(provider.Variants).To(HaveLen(2))
			Expect(provider.Active).To(BeTrue())
		})

		It("derives a deterministic top-level endpoint for the derived fields", func() {
			preset, err := FindPreset("minimax")
			Expect(err).NotTo(HaveOccurred())
			provider := ProviderFromPreset(*preset, "")
			// UpsertProvider derives these; the preset projection itself keeps
			// them empty so the store decides the canonical default.
			Expect(provider.BaseURL).To(BeEmpty())
			Expect(provider.APIProtocol).To(BeEmpty())
		})

		It("falls back to the first model as default when default_model is absent", func() {
			provider := ProviderFromPreset(ProviderPreset{
				Slug: "custom", Name: "Custom",
				Models:   []string{"m1", "m2"},
				Variants: []PresetVariant{{Protocol: "openai_chat", BaseURL: "https://custom.test/v1"}},
			}, "")
			Expect(provider.DefaultModel).To(Equal("m1"))
		})

		It("keeps claude capabilities on the anthropic variant only", func() {
			preset, _ := FindPreset("glm")
			provider := ProviderFromPreset(*preset, "k")
			Expect(provider.Variants["anthropic"].Capabilities).To(HaveKey("claude_extra_env"))
			Expect(provider.Variants["openai_chat"].Capabilities).To(BeEmpty())
		})
	})

	Describe("PresetLabel", func() {
		It("summarizes protocols and models", func() {
			preset, _ := FindPreset("minimax")
			label := PresetLabel(*preset)
			Expect(label).To(ContainSubstring("MiniMax"))
			Expect(label).To(ContainSubstring("anthropic"))
			Expect(label).To(ContainSubstring("MiniMax-M3"))
		})
	})
})

var _ = Describe("config example template", func() {
	It("embeds a readable example config", func() {
		content, err := ConfigExample()
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(ContainSubstring("[[providers]]"))
	})
})

var _ = Describe("store integration of preset projection", func() {
	It("produces a provider that HasModel accepts for every bundled model", func() {
		preset, _ := FindPreset("glm")
		provider := ProviderFromPreset(*preset, "")
		var storeProvider store.Provider = provider
		Expect(storeProvider.HasModel("glm-5.3")).To(BeTrue())
		Expect(storeProvider.HasModel("glm-9")).To(BeFalse())
	})
})

var _ = Describe("user preset files", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		os.Setenv("AISW_PRESETS_DIR", dir)
		DeferCleanup(func() { os.Unsetenv("AISW_PRESETS_DIR") })
	})

	It("starts with only builtin presets", func() {
		all, err := AllPresets()
		Expect(err).NotTo(HaveOccurred())
		Expect(all).To(HaveLen(7))
		for _, preset := range all {
			Expect(preset.Source).To(Equal("builtin"))
		}
	})

	It("saves a provider as a preset file and reloads it", func() {
		provider := store.Provider{
			Slug: "custom", Name: "Custom Vendor",
			DefaultModel: "m1", Models: []string{"m1", "m2"},
			Variants: map[string]store.ProviderVariant{
				"anthropic":   {BaseURL: "https://a.test"},
				"openai_chat": {BaseURL: "https://c.test/v1", Endpoints: map[string]string{"chat_completions": "/chat/completions"}},
			},
		}
		path, err := SaveUserPreset(PresetFromProvider(provider))
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(HaveSuffix("custom.toml"))

		// persisted file must not contain an API key
		content, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).NotTo(ContainSubstring("api_key"))

		all, err := AllPresets()
		Expect(err).NotTo(HaveOccurred())
		var found *SourcedPreset
		for i := range all {
			if all[i].Slug == "custom" {
				found = &all[i]
			}
		}
		Expect(found).NotTo(BeNil())
		Expect(found.Source).To(Equal("user"))
		Expect(found.Models).To(ConsistOf("m1", "m2"))
		Expect(found.Variants).To(HaveLen(2))

		// the reloaded preset projects back into a working provider
		projected := ProviderFromPreset(found.ProviderPreset, "key")
		Expect(projected.Variants).To(HaveLen(2))
		Expect(projected.DefaultModel).To(Equal("m1"))
	})

	It("lets user presets override builtin entries of the same slug", func() {
		preset := ProviderPreset{
			Slug: "glm", Name: "GLM Override",
			Models:   []string{"glm-9"},
			Variants: []PresetVariant{{Protocol: "anthropic", BaseURL: "https://override.test"}},
		}
		_, err := SaveUserPreset(preset)
		Expect(err).NotTo(HaveOccurred())

		found, err := FindSourcedPreset("glm")
		Expect(err).NotTo(HaveOccurred())
		Expect(found.Source).To(Equal("user"))
		Expect(found.Name).To(Equal("GLM Override"))
	})

	It("imports presets from an external TOML file", func() {
		path := filepath.Join(dir, "incoming.toml")
		Expect(os.WriteFile(path, []byte(`
[[presets]]
slug = "imported"
name = "Imported Vendor"
models = ["x1"]

  [[presets.variants]]
  protocol = "openai_chat"
  base_url = "https://imported.test/v1"
`), 0o600)).To(Succeed())

		imported, err := ImportUserPresetFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(imported).To(HaveLen(1))

		found, err := FindSourcedPreset("imported")
		Expect(err).NotTo(HaveOccurred())
		Expect(found.Source).To(Equal("user"))
		Expect(found.Variants).To(HaveLen(1))
	})

	It("deletes user presets but refuses builtin ones", func() {
		Expect(DeleteUserPreset("glm")).To(MatchError(ContainSubstring("user preset not found")))

		_, err := SaveUserPreset(ProviderPreset{
			Slug: "tmp", Name: "Tmp",
			Variants: []PresetVariant{{Protocol: "openai_chat", BaseURL: "https://t.test"}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(DeleteUserPreset("tmp")).To(Succeed())

		_, err = FindSourcedPreset("tmp")
		Expect(err).To(MatchError(ContainSubstring("not found")))
	})

	It("derives a one-variant preset from a legacy provider", func() {
		legacy := store.Provider{
			Slug: "legacy", BaseURL: "https://l.test/v1",
			APIProtocol: "openai_chat", DefaultModel: "lm",
		}
		preset := PresetFromProvider(legacy)
		Expect(preset.Variants).To(HaveLen(1))
		Expect(preset.Variants[0].Protocol).To(Equal("openai_chat"))
	})
})
