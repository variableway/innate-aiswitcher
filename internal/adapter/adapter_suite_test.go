package adapter

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

func TestAdapterSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Adapter Suite")
}

// glmVendor mirrors the bundled GLM preset stored as one vendor row.
func glmVendor() store.Provider {
	return store.Provider{
		Slug:         "glm",
		Name:         "GLM (Volcengine Ark)",
		APIKey:       "ark-one-key",
		APIProtocol:  "openai_chat",
		BaseURL:      "https://ark.cn-beijing.volces.com/api/v3",
		DefaultModel: "glm-5.2",
		Models:       []string{"glm-5.2", "glm-5.3"},
		Variants: map[string]store.ProviderVariant{
			"anthropic": {
				BaseURL:   "https://ark.cn-beijing.volces.com/api/plan",
				Endpoints: map[string]string{"messages": "/v1/messages", "models": "/v1/models"},
				Capabilities: map[string]interface{}{
					"claude_extra_env": map[string]interface{}{
						"API_TIMEOUT_MS": "3000000",
					},
				},
			},
			"openai_chat": {
				BaseURL:   "https://ark.cn-beijing.volces.com/api/v3",
				Endpoints: map[string]string{"chat_completions": "/chat/completions", "models": "/models"},
			},
		},
	}
}

var _ = Describe("launch plans from a vendor provider", func() {
	var provider store.Provider

	BeforeEach(func() {
		provider = glmVendor()
	})

	readClaudeSettings := func(plan LaunchPlan) map[string]any {
		bytes, err := os.ReadFile(plan.Files["settings"])
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		var settings map[string]any
		ExpectWithOffset(1, json.Unmarshal(bytes, &settings)).To(Succeed())
		return settings
	}

	It("sends claude to the anthropic variant with the shared key", func() {
		plan, cleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, provider, nil, LaunchOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		settings := readClaudeSettings(plan)
		env := settings["env"].(map[string]any)
		Expect(env["ANTHROPIC_BASE_URL"]).To(Equal("https://ark.cn-beijing.volces.com/api/plan"))
		Expect(env["ANTHROPIC_AUTH_TOKEN"]).To(Equal("ark-one-key"))
		Expect(env["ANTHROPIC_MODEL"]).To(Equal("glm-5.2"))
		Expect(env["API_TIMEOUT_MS"]).To(Equal("3000000"), "variant capabilities must reach claude")
	})

	It("sends codex to a chat endpoint and writes an ephemeral CODEX_HOME", func() {
		plan, cleanup, err := BuildPlan(store.Agent{Binary: "codex", Adapter: "codex"}, provider, nil, LaunchOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		Expect(plan.Env["CODEX_HOME"]).NotTo(BeEmpty())
		config, err := os.ReadFile(plan.Files["config"])
		Expect(err).NotTo(HaveOccurred())
		text := string(config)
		Expect(text).To(ContainSubstring(`wire_api = "chat"`), "codex falls back to chat when the vendor has no responses endpoint")
		Expect(text).To(ContainSubstring(`base_url = "https://ark.cn-beijing.volces.com/api/v3"`))
		Expect(text).NotTo(ContainSubstring("ark-one-key"), "chat wire api authenticates via auth.json, not the bearer config")
		auth, err := os.ReadFile(plan.Files["auth"])
		Expect(err).NotTo(HaveOccurred())
		Expect(string(auth)).To(ContainSubstring("ark-one-key"))
	})

	It("sends opencode out with OPENAI_* env from the chat variant", func() {
		plan, cleanup, err := BuildPlan(store.Agent{Binary: "opencode", Adapter: "openai_env"}, provider, nil, LaunchOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		Expect(plan.Env["OPENAI_API_KEY"]).To(Equal("ark-one-key"))
		Expect(plan.Env["OPENAI_BASE_URL"]).To(Equal("https://ark.cn-beijing.volces.com/api/v3"))
	})

	Describe("model selection", func() {
		It("launches a newly added model that shares the vendor key", func() {
			provider.Models = append(provider.Models, "glm-5.3")
			plan, cleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, provider, nil, LaunchOptions{Model: "glm-5.3"})
			Expect(err).NotTo(HaveOccurred())
			defer cleanup()

			settings := readClaudeSettings(plan)
			env := settings["env"].(map[string]any)
			Expect(env["ANTHROPIC_MODEL"]).To(Equal("glm-5.3"))
			Expect(env["ANTHROPIC_AUTH_TOKEN"]).To(Equal("ark-one-key"), "same key, no reconfiguration")
		})

		It("prefers the launch --model over profile and provider defaults", func() {
			profile := &store.Profile{Model: "glm-5.2"}
			plan, cleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, provider, profile, LaunchOptions{Model: "glm-5.3"})
			Expect(err).NotTo(HaveOccurred())
			defer cleanup()

			settings := readClaudeSettings(plan)
			Expect(settings["env"].(map[string]any)["ANTHROPIC_MODEL"]).To(Equal("glm-5.3"))
		})

		It("rejects a model that is not configured on the provider", func() {
			_, cleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, provider, nil, LaunchOptions{Model: "glm-9"})
			if cleanup != nil {
				cleanup()
			}
			Expect(err).To(MatchError(ContainSubstring("aisw provider model add glm glm-9")))
		})

		It("still resolves the profile model override when no flag is given", func() {
			profile := &store.Profile{Model: "glm-5.3"}
			plan, cleanup, err := BuildPlan(store.Agent{Binary: "codex", Adapter: "codex"}, provider, profile, LaunchOptions{})
			Expect(err).NotTo(HaveOccurred())
			defer cleanup()

			config, err := os.ReadFile(plan.Files["config"])
			Expect(err).NotTo(HaveOccurred())
			Expect(string(config)).To(ContainSubstring(`model = "glm-5.3"`))
		})
	})

	Describe("agent/protocol mismatches", func() {
		It("refuses a provider whose only endpoint cannot serve the agent", func() {
			legacy := store.Provider{Slug: "chat-only", BaseURL: "https://x.test/v1", APIProtocol: "openai_chat", DefaultModel: "m"}
			_, cleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, legacy, nil, LaunchOptions{})
			if cleanup != nil {
				cleanup()
			}
			Expect(err).To(MatchError(ContainSubstring("does not support agent claude")))
		})

		It("rejects adapters outside the focused set", func() {
			_, cleanup, err := BuildPlan(store.Agent{Binary: "gemini", Adapter: "gemini"}, provider, nil, LaunchOptions{})
			if cleanup != nil {
				cleanup()
			}
			Expect(err).To(MatchError(ContainSubstring("unsupported adapter")))
		})
	})

	Describe("dry-run execution", func() {
		It("prints the plan without launching anything", func() {
			plan, cleanup, err := BuildPlan(store.Agent{Binary: "claude", Adapter: "claude"}, provider, nil, LaunchOptions{})
			Expect(err).NotTo(HaveOccurred())
			defer cleanup()
			Expect(Execute(plan, func() {}, LaunchOptions{DryRun: true})).To(Succeed())
		})
	})

	It("keeps skip-permissions behavior for vendor providers", func() {
		agent := store.Agent{Binary: "claude", Adapter: "claude", SkipPermissionsArg: "--dangerously-skip-permissions", SkipPermissionsDefault: true}
		plan, cleanup, err := BuildPlan(agent, provider, nil, LaunchOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer cleanup()
		Expect(plan.Command).To(ContainSubstring("--dangerously-skip-permissions"))
	})
})
