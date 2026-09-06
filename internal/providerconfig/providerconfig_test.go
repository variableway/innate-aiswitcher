package providerconfig_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/providerconfig"
	"github.com/variableway/innate-aiswitcher/internal/store"
)

// minimaxVendor mirrors the bundled MiniMax preset: one vendor row, one API
// key, three protocol variants serving every focused agent.
func minimaxVendor() store.Provider {
	return store.Provider{
		Slug:         "minimax",
		Name:         "MiniMax",
		APIKey:       "sk-minimax-once",
		APIProtocol:  "openai_chat",
		BaseURL:      "https://api.minimaxi.com/v1",
		DefaultModel: "MiniMax-M3",
		Models:       []string{"MiniMax-M3", "MiniMax-M2.5"},
		Variants: map[string]store.ProviderVariant{
			"anthropic": {
				BaseURL:   "https://api.minimaxi.com/anthropic",
				Endpoints: map[string]string{"messages": "/v1/messages", "models": "/v1/models"},
				Capabilities: map[string]interface{}{
					"claude_extra_env": map[string]interface{}{"API_TIMEOUT_MS": "3000000"},
				},
			},
			"openai_responses": {
				BaseURL:   "https://api.minimaxi.com/v1",
				Endpoints: map[string]string{"responses": "/responses", "models": "/models"},
				Capabilities: map[string]interface{}{
					"codex_auth_mode": "experimental_bearer_token",
				},
			},
			"openai_chat": {
				BaseURL:   "https://api.minimaxi.com/v1",
				Endpoints: map[string]string{"chat_completions": "/chat/completions", "models": "/models"},
			},
		},
	}
}

var _ = Describe("LLM Provider Config abstraction", func() {
	Describe("one vendor with one API key serving every agent", func() {
		It("resolves the anthropic variant for the claude adapter", func() {
			resolved, err := providerconfig.Resolve(minimaxVendor(), "claude")
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved.APIProtocol).To(Equal("anthropic"))
			Expect(resolved.BaseURL).To(Equal("https://api.minimaxi.com/anthropic"))
			Expect(resolved.Endpoints).To(HaveKeyWithValue("messages", "/v1/messages"))
			Expect(resolved.APIKey).To(Equal("sk-minimax-once"), "the vendor key must be shared, not duplicated")
			Expect(resolved.Models).To(ConsistOf("MiniMax-M3", "MiniMax-M2.5"))
			Expect(resolved.Capabilities).To(HaveKey("claude_extra_env"))
		})

		It("resolves the responses variant for the codex adapter", func() {
			resolved, err := providerconfig.Resolve(minimaxVendor(), "codex")
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved.APIProtocol).To(Equal("openai_responses"))
			Expect(resolved.BaseURL).To(Equal("https://api.minimaxi.com/v1"))
			Expect(resolved.Capabilities).To(HaveKeyWithValue("codex_auth_mode", "experimental_bearer_token"))
			Expect(resolved.APIKey).To(Equal("sk-minimax-once"))
		})

		It("resolves the chat variant for the opencode adapter", func() {
			resolved, err := providerconfig.Resolve(minimaxVendor(), "openai_env")
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved.APIProtocol).To(Equal("openai_chat"))
			Expect(resolved.BaseURL).To(Equal("https://api.minimaxi.com/v1"))
			Expect(resolved.Endpoints).To(HaveKeyWithValue("chat_completions", "/chat/completions"))
		})

		It("lets codex fall back to chat when a vendor has no responses endpoint", func() {
			glm := store.Provider{
				Slug:   "glm",
				APIKey: "ark-key",
				Variants: map[string]store.ProviderVariant{
					"anthropic":   {BaseURL: "https://ark.example/api/plan"},
					"openai_chat": {BaseURL: "https://ark.example/api/v3"},
				},
			}
			resolved, err := providerconfig.Resolve(glm, "codex")
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved.APIProtocol).To(Equal("openai_chat"))
			Expect(resolved.BaseURL).To(Equal("https://ark.example/api/v3"))
		})

		It("errors when the vendor has no endpoint the agent can speak", func() {
			anthropicOnly := store.Provider{
				Slug:     "anthropic-only",
				Variants: map[string]store.ProviderVariant{"anthropic": {BaseURL: "https://example.test"}},
			}
			_, err := providerconfig.Resolve(anthropicOnly, "openai_env")
			Expect(err).To(MatchError(ContainSubstring("no endpoint for agent openai_env")))
		})
	})

	Describe("newly added models keep working across agents", func() {
		It("accepts any model when the provider carries a model list containing it", func() {
			vendor := minimaxVendor()
			vendor.Models = append(vendor.Models, "MiniMax-M4")
			for _, adapter := range []string{"claude", "codex", "openai_env"} {
				resolved, err := providerconfig.Resolve(vendor, adapter)
				Expect(err).NotTo(HaveOccurred())
				Expect(resolved.HasModel("MiniMax-M4")).To(BeTrue())
				Expect(resolved.APIKey).To(Equal("sk-minimax-once"), "adding a model must not require another key")
			}
		})
	})

	Describe("legacy single-protocol providers", func() {
		It("passes through unchanged when the adapter speaks the protocol", func() {
			legacy := store.Provider{
				Slug:        "minimax-openai",
				APIKey:      "sk-old",
				APIProtocol: "openai_chat",
				BaseURL:     "https://api.minimaxi.com/v1",
			}
			resolved, err := providerconfig.Resolve(legacy, "codex")
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(Equal(legacy))
		})

		It("errors when the protocol cannot serve the agent", func() {
			legacy := store.Provider{Slug: "openai-only", APIProtocol: "openai_chat", BaseURL: "https://api.openai.com/v1"}
			_, err := providerconfig.Resolve(legacy, "claude")
			Expect(err).To(MatchError(ContainSubstring("does not support agent claude")))
		})
	})

	Describe("protocol-agnostic consumers", func() {
		It("ResolveDefault keeps the top-level endpoint when already set", func() {
			resolved := providerconfig.ResolveDefault(minimaxVendor())
			Expect(resolved.APIProtocol).To(Equal("openai_chat"))
			Expect(resolved.BaseURL).To(Equal("https://api.minimaxi.com/v1"))
		})

		It("ResolveDefault picks a variant when top-level fields are missing", func() {
			vendor := minimaxVendor()
			vendor.APIProtocol = ""
			vendor.BaseURL = ""
			resolved := providerconfig.ResolveDefault(vendor)
			Expect(resolved.APIProtocol).To(Equal("openai_chat"))
			Expect(resolved.BaseURL).To(Equal("https://api.minimaxi.com/v1"))
		})
	})

	Describe("SupportsAgent", func() {
		It("accepts variant vendors and matching legacy providers", func() {
			Expect(providerconfig.SupportsAgent(minimaxVendor(), "claude")).To(BeTrue())
			Expect(providerconfig.SupportsAgent(minimaxVendor(), "codex")).To(BeTrue())
			Expect(providerconfig.SupportsAgent(minimaxVendor(), "openai_env")).To(BeTrue())
			legacy := store.Provider{Slug: "legacy", APIProtocol: "anthropic"}
			Expect(providerconfig.SupportsAgent(legacy, "claude")).To(BeTrue())
			Expect(providerconfig.SupportsAgent(legacy, "codex")).To(BeFalse())
		})
	})

	Describe("agent protocol registry", func() {
		It("maps every focused agent to its wire protocols", func() {
			claudeProtocols, ok := providerconfig.AgentProtocols("claude")
			Expect(ok).To(BeTrue())
			Expect(claudeProtocols).To(Equal([]string{"anthropic"}))
			codexProtocols, ok := providerconfig.AgentProtocols("codex")
			Expect(ok).To(BeTrue())
			Expect(codexProtocols).To(Equal([]string{"openai_responses", "openai_chat"}))
			envProtocols, ok := providerconfig.AgentProtocols("openai_env")
			Expect(ok).To(BeTrue())
			Expect(envProtocols).To(Equal([]string{"openai_chat"}))
			_, ok = providerconfig.AgentProtocols("gemini")
			Expect(ok).To(BeFalse(), "non-focused adapters must not resolve")
		})

		It("reports which agents a protocol serves", func() {
			Expect(providerconfig.SupportedAgents("anthropic")).To(Equal([]string{"claude"}))
			Expect(providerconfig.SupportedAgents("openai_chat")).To(ConsistOf("codex", "openai_env"))
		})
	})
})

var _ = Describe("provider model list semantics", func() {
	It("accepts any model when no explicit list is configured", func() {
		Expect(store.Provider{Slug: "x"}.HasModel("anything")).To(BeTrue())
	})
	It("restricts models to the configured list", func() {
		p := store.Provider{Slug: "x", Models: []string{"glm-5.2", "glm-5.3"}}
		Expect(p.HasModel("glm-5.3")).To(BeTrue())
		Expect(p.HasModel("glm-6")).To(BeFalse())
	})
	It("preserves provider identity fields through resolution", func() {
		vendor := minimaxVendor()
		vendor.ID = "rec-123"
		resolved, err := providerconfig.Resolve(vendor, "claude")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.ID).To(Equal("rec-123"))
		Expect(resolved.Slug).To(Equal("minimax"))
		Expect(resolved.DefaultModel).To(Equal("MiniMax-M3"))
	})
})
