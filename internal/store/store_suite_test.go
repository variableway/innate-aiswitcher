package store

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"

	_ "github.com/variableway/innate-aiswitcher/migrations"
)

func TestStoreSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Store Suite")
}

// newTestStore bootstraps a real PocketBase instance with all migrations
// applied against a throwaway data dir, mirroring the production wiring in
// internal/app.
func newTestStore() *Store {
	dir := GinkgoT().TempDir()
	pb := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dir})
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
	ExpectWithOffset(1, pb.Bootstrap()).To(Succeed())
	return New(pb)
}

// glmVendorRow mirrors what templates.ProviderFromPreset produces for the GLM
// preset: one row, one key, anthropic + openai_chat variants, model list.
func glmVendorRow() Provider {
	return Provider{
		Slug:         "glm",
		Name:         "GLM (Volcengine Ark)",
		APIKey:       "ark-single-key",
		DefaultModel: "glm-5.2",
		Models:       []string{"glm-5.2"},
		Variants: map[string]ProviderVariant{
			"anthropic": {
				BaseURL:   "https://ark.cn-beijing.volces.com/api/plan",
				Endpoints: map[string]string{"messages": "/v1/messages", "models": "/v1/models"},
				Capabilities: map[string]interface{}{
					"claude_extra_env": map[string]interface{}{"API_TIMEOUT_MS": "3000000"},
				},
			},
			"openai_chat": {
				BaseURL:   "https://ark.cn-beijing.volces.com/api/v3",
				Endpoints: map[string]string{"chat_completions": "/chat/completions", "models": "/models"},
			},
		},
	}
}

var _ = Describe("vendor provider storage", func() {
	var s *Store

	BeforeEach(func() {
		s = newTestStore()
	})

	Describe("focused agent catalog", func() {
		It("seeds exactly claude, codex and opencode after migrations", func() {
			agents, err := s.ListAgents()
			Expect(err).NotTo(HaveOccurred())
			slugs := make([]string, 0, len(agents))
			for _, agent := range agents {
				slugs = append(slugs, agent.Slug)
			}
			Expect(slugs).To(ConsistOf("claude", "codex", "opencode"), "gemini/kimi/trae/hermes/openclaw must be removed")
		})
	})

	Describe("persisting a vendor provider", func() {
		It("round-trips variants, models and the single API key", func() {
			saved, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.Slug).To(Equal("glm"))

			loaded, err := s.GetProvider("glm")
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.APIKey).To(Equal("ark-single-key"))
			Expect(loaded.Models).To(Equal([]string{"glm-5.2"}))
			Expect(loaded.Variants).To(HaveLen(2))
			Expect(loaded.Variants["anthropic"].BaseURL).To(Equal("https://ark.cn-beijing.volces.com/api/plan"))
			Expect(loaded.Variants["anthropic"].Capabilities).To(HaveKey("claude_extra_env"))
			Expect(loaded.Variants["openai_chat"].Endpoints).To(HaveKeyWithValue("chat_completions", "/chat/completions"))
		})

		It("derives a top-level base_url and api_protocol from the preferred variant", func() {
			saved, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.BaseURL).To(Equal("https://ark.cn-beijing.volces.com/api/v3"))
			Expect(saved.APIProtocol).To(Equal("openai_chat"))
		})

		It("keeps an existing API key when upserting without one", func() {
			_, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())

			update := glmVendorRow()
			update.APIKey = ""
			update.Name = "GLM renamed"
			saved, err := s.UpsertProvider(update)
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.APIKey).To(Equal("ark-single-key"), "empty key must preserve the stored key (Web PUT semantics)")
			Expect(saved.Name).To(Equal("GLM renamed"))
		})
	})

	Describe("adding models to a vendor", func() {
		It("shares the vendor API key with the new model", func() {
			_, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())

			saved, err := s.AddModel("glm", "glm-5.3", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.Models).To(ConsistOf("glm-5.2", "glm-5.3"))
			Expect(saved.APIKey).To(Equal("ark-single-key"), "adding a model must reuse the key")
			Expect(saved.DefaultModel).To(Equal("glm-5.2"), "default stays unless requested")

			loaded, _ := s.GetProvider("glm")
			Expect(loaded.Models).To(ConsistOf("glm-5.2", "glm-5.3"))
			Expect(loaded.APIKey).To(Equal("ark-single-key"))
		})

		It("can promote the new model to default", func() {
			_, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())

			saved, err := s.AddModel("glm", "glm-5.3", true)
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.DefaultModel).To(Equal("glm-5.3"))
		})

		It("is idempotent for a model that already exists", func() {
			_, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())

			saved, err := s.AddModel("glm", "glm-5.2", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.Models).To(Equal([]string{"glm-5.2"}))
		})
	})

	Describe("removing models", func() {
		It("re-points the default to the first remaining model", func() {
			row := glmVendorRow()
			row.Models = []string{"glm-5.2", "glm-5.3"}
			_, err := s.UpsertProvider(row)
			Expect(err).NotTo(HaveOccurred())

			saved, err := s.RemoveModel("glm", "glm-5.2")
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.Models).To(Equal([]string{"glm-5.3"}))
			Expect(saved.DefaultModel).To(Equal("glm-5.3"))
		})

		It("keeps state unchanged for unknown models", func() {
			_, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())

			saved, err := s.RemoveModel("glm", "glm-9")
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.Models).To(Equal([]string{"glm-5.2"}))
			Expect(saved.DefaultModel).To(Equal("glm-5.2"))
		})
	})

	Describe("selecting a vendor for a session", func() {
		It("resolves the vendor provider for every focused agent", func() {
			_, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())

			for _, agentSlug := range []string{"claude", "codex", "opencode"} {
				agent, provider, profile, err := s.ResolveSelector(agentSlug, "glm")
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				ExpectWithOffset(1, agent.Slug).To(Equal(agentSlug))
				ExpectWithOffset(1, provider.Slug).To(Equal("glm"))
				ExpectWithOffset(1, provider.APIKey).To(Equal("ark-single-key"))
				ExpectWithOffset(1, profile).To(BeNil())
			}
		})

		It("supports profiles that pin a vendor model", func() {
			_, err := s.UpsertProvider(glmVendorRow())
			Expect(err).NotTo(HaveOccurred())
			_, err = s.AddModel("glm", "glm-5.3", false)
			Expect(err).NotTo(HaveOccurred())

			profile, err := s.UpsertProfile(Profile{
				Slug: "claude-glm53", AgentSlug: "claude", ProviderSlug: "glm",
				Model: "glm-5.3", IsDefault: true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(profile.Model).To(Equal("glm-5.3"))

			_, provider, resolved, err := s.ResolveSelector("claude", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved.Slug).To(Equal("claude-glm53"))
			Expect(provider.Models).To(ConsistOf("glm-5.2", "glm-5.3"))
		})
	})

	Describe("legacy sibling providers keep working", func() {
		It("still stores single-protocol rows and syncs their keys", func() {
			_, err := s.UpsertProvider(Provider{
				Slug: "minimax-claude", Name: "MiniMax claude", BaseURL: "https://api.minimaxi.com/anthropic",
				APIProtocol: "anthropic", DefaultModel: "MiniMax-M3", APIKey: "sk-shared-legacy",
			})
			Expect(err).NotTo(HaveOccurred())

			loaded, err := s.GetProvider("minimax-claude")
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.APIKey).To(Equal("sk-shared-legacy"))
			Expect(loaded.Variants).To(BeEmpty())
		})
	})
})

var _ = Describe("legacy vendor normalization", func() {
	var s *Store

	minimaxPreset := func() Provider {
		return Provider{
			Slug: "minimax", Name: "MiniMax",
			DefaultModel: "MiniMax-M3", Models: []string{"MiniMax-M3"},
			Variants: map[string]ProviderVariant{
				"anthropic":       {BaseURL: "https://api.minimaxi.com/anthropic"},
				"openai_responses": {BaseURL: "https://api.minimaxi.com/v1"},
				"openai_chat":      {BaseURL: "https://api.minimaxi.com/v1"},
			},
		}
	}

	BeforeEach(func() {
		s = newTestStore()
	})

	It("merges minimax-claude/minimax-codex into a single MiniMax row", func() {
		_, err := s.UpsertProvider(Provider{
			Slug: "minimax-claude", Name: "MiniMax Claude Code-compatible",
			BaseURL: "https://api.minimaxi.com/anthropic", APIProtocol: "anthropic",
			DefaultModel: "MiniMax-M3", APIKey: "sk-legacy-key",
		})
		Expect(err).NotTo(HaveOccurred())
		_, err = s.UpsertProvider(Provider{
			Slug: "minimax-codex", Name: "MiniMax Codex-compatible",
			BaseURL: "https://api.minimaxi.com/v1", APIProtocol: "openai_responses",
			DefaultModel: "MiniMax-M3",
		})
		Expect(err).NotTo(HaveOccurred())
		_, err = s.UpsertProfile(Profile{Slug: "claude-mm", AgentSlug: "claude", ProviderSlug: "minimax-claude"})
		Expect(err).NotTo(HaveOccurred())

		merged, err := s.NormalizeLegacyVendors(map[string]Provider{"minimax": minimaxPreset()})
		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(ContainElement("minimax"))

		providers, err := s.ListProviders()
		Expect(err).NotTo(HaveOccurred())
		Expect(providers).To(HaveLen(1))
		Expect(providers[0].Slug).To(Equal("minimax"))
		Expect(providers[0].Name).To(Equal("MiniMax"))
		Expect(providers[0].APIKey).To(Equal("sk-legacy-key"), "the key must survive the merge")
		Expect(providers[0].Variants).To(HaveLen(3))

		// the profile now points at the merged row
		profile, err := s.GetProfile("claude-mm")
		Expect(err).NotTo(HaveOccurred())
		Expect(profile.ProviderSlug).To(Equal("minimax"))
	})

	It("keeps user-configured models and default across the merge", func() {
		_, err := s.UpsertProvider(Provider{
			Slug: "volcengine-claude", Name: "Volcengine Ark (火山方舟) Claude Code-compatible",
			BaseURL: "https://ark.cn-beijing.volces.com/api/plan", APIProtocol: "anthropic",
			DefaultModel: "glm-5.2", Models: []string{"glm-5.2", "glm-5.3-custom"}, APIKey: "ark-key",
		})
		Expect(err).NotTo(HaveOccurred())

		glmPreset := Provider{
			Slug: "glm", Name: "GLM (Volcengine Ark)", DefaultModel: "glm-5.2",
			Models: []string{"glm-5.2", "glm-5.3"},
			Variants: map[string]ProviderVariant{
				"anthropic":  {BaseURL: "https://ark.cn-beijing.volces.com/api/plan"},
				"openai_chat": {BaseURL: "https://ark.cn-beijing.volces.com/api/v3"},
			},
		}
		merged, err := s.NormalizeLegacyVendors(map[string]Provider{"glm": glmPreset})
		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(ContainElement("glm"))

		provider, err := s.GetProvider("glm")
		Expect(err).NotTo(HaveOccurred())
		Expect(provider.Name).To(Equal("GLM (Volcengine Ark)"))
		Expect(provider.Models).To(ConsistOf("glm-5.2", "glm-5.3", "glm-5.3-custom"))
		Expect(provider.DefaultModel).To(Equal("glm-5.2"))
	})

	It("is idempotent and leaves clean vendor rows alone", func() {
		_, err := s.UpsertProvider(minimaxPreset())
		Expect(err).NotTo(HaveOccurred())

		merged, err := s.NormalizeLegacyVendors(map[string]Provider{"minimax": minimaxPreset()})
		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(BeEmpty())

		providers, _ := s.ListProviders()
		Expect(providers).To(HaveLen(1))
	})

	It("merges even without a matching preset, keeping the legacy endpoint", func() {
		_, err := s.UpsertProvider(Provider{
			Slug: "myvendor-openai", Name: "MyVendor OpenAI-compatible",
			BaseURL: "https://my.example/v1", APIProtocol: "openai_chat",
			DefaultModel: "m1", APIKey: "k",
		})
		Expect(err).NotTo(HaveOccurred())

		// no preset for "myvendor" — normalization leaves unknown groups alone
		merged, err := s.NormalizeLegacyVendors(map[string]Provider{})
		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(BeEmpty())
		provider, err := s.GetProvider("myvendor-openai")
		Expect(err).NotTo(HaveOccurred())
		Expect(provider.APIKey).To(Equal("k"))
	})
})
