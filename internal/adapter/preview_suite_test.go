package adapter

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

// vendor mirrors a stored GLM provider row: one key, two variants.
func vendor() store.Provider {
	return store.Provider{
		Slug: "glm", Name: "GLM", APIKey: "sk-preview",
		DefaultModel: "glm-5.2", Models: []string{"glm-5.2", "glm-5.3"},
		Variants: map[string]store.ProviderVariant{
			"anthropic":   {BaseURL: "https://ark.example/api/plan"},
			"openai_chat": {BaseURL: "https://ark.example/api/v3"},
		},
	}
}

var _ = Describe("config preview per agent/provider/model", func() {
	It("projects claude onto a writable settings.json", func() {
		result, err := Preview(store.Agent{Slug: "claude", Adapter: "claude", Binary: "claude"}, vendor(), "glm-5.3")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Model).To(Equal("glm-5.3"))
		Expect(result.Files).To(HaveLen(1))
		file := result.Files[0]
		Expect(file.Name).To(Equal("settings"))
		Expect(file.Label).To(Equal("settings.json"))
		Expect(file.Writable).To(BeTrue())
		Expect(file.Content).To(ContainSubstring(`"ANTHROPIC_MODEL": "glm-5.3"`))
		Expect(file.Content).To(ContainSubstring("sk-preview"))
		Expect(file.Content).To(ContainSubstring("https://ark.example/api/plan"))
	})

	It("projects codex onto config.toml plus auth.json", func() {
		result, err := Preview(store.Agent{Slug: "codex", Adapter: "codex", Binary: "codex"}, vendor(), "glm-5.2")
		Expect(err).NotTo(HaveOccurred())
		labels := []string{}
		for _, file := range result.Files {
			labels = append(labels, file.Label)
		}
		Expect(labels).To(ContainElements("config.toml", "auth.json", "CODEX_HOME (temp dir)"))
		Expect(result.Env).To(HaveKey("CODEX_HOME"))
		for _, file := range result.Files {
			if file.Label == "config.toml" {
				Expect(file.Content).To(ContainSubstring(`wire_api = "chat"`))
				Expect(file.Writable).To(BeTrue())
			}
		}
	})

	It("projects opencode onto env vars only", func() {
		result, err := Preview(store.Agent{Slug: "opencode", Adapter: "openai_env", Binary: "opencode"}, vendor(), "glm-5.2")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Files).To(BeEmpty())
		Expect(result.Env["OPENAI_API_KEY"]).To(Equal("sk-preview"))
		Expect(result.Env["OPENAI_BASE_URL"]).To(Equal("https://ark.example/api/v3"))
	})

	It("shares the vendor key across every agent and model", func() {
		for _, adapterName := range []string{"claude", "codex", "openai_env"} {
			for _, model := range []string{"glm-5.2", "glm-5.3"} {
				result, err := Preview(store.Agent{Slug: "x", Adapter: adapterName, Binary: "x"}, vendor(), model)
				Expect(err).NotTo(HaveOccurred())
				blob := strings.Join(mapValues(result.Env), "\n")
				for _, file := range result.Files {
					blob += "\n" + file.Content
				}
				Expect(blob).To(ContainSubstring("sk-preview"))
				Expect(result.Model).To(Equal(model))
			}
		}
	})

	It("rejects unconfigured models with the model-add hint", func() {
		_, err := Preview(store.Agent{Slug: "claude", Adapter: "claude", Binary: "claude"}, vendor(), "glm-9")
		Expect(err).To(MatchError(ContainSubstring("provider model add")))
	})
})

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
