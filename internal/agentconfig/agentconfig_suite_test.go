package agentconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/agentconfig"
)

func TestAgentConfigSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AgentConfig Suite")
}

var _ = Describe("agent configuration files", func() {
	var home string

	BeforeEach(func() {
		home = GinkgoT().TempDir()
		original := os.Getenv("HOME")
		os.Setenv("HOME", home)
		DeferCleanup(func() {
			if original == "" {
				os.Unsetenv("HOME")
			} else {
				os.Setenv("HOME", original)
			}
		})
	})

	It("lists every agent with its whitelisted files and content", func() {
		agents, err := agentconfig.List()
		Expect(err).NotTo(HaveOccurred())
		Expect(agents).To(HaveLen(3))
		Expect(agents[0].Agent).To(Equal("claude"))
		Expect(agents[0].Name).To(Equal("Claude Code"))
		Expect(agents[0].Files).To(HaveLen(1))
		Expect(agents[1].Files).To(HaveLen(2)) // config.toml + auth.json
		Expect(agents[2].Files).To(HaveLen(1))
		for _, agent := range agents {
			for _, file := range agent.Files {
				Expect(file.Editable).To(BeTrue())
				Expect(file.Exists).To(BeFalse(), "fresh home has no agent configs")
			}
		}
	})

	It("reads a file that exists on disk", func() {
		path := filepath.Join(home, ".claude")
		Expect(os.MkdirAll(path, 0o700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(path, "settings.json"), []byte(`{"env":{}}`), 0o600)).To(Succeed())

		info, err := agentconfig.Read("claude", "settings")
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Exists).To(BeTrue())
		Expect(info.Content).To(ContainSubstring(`"env"`))
	})

	It("writes atomically and creates parent directories", func() {
		info, err := agentconfig.Write("codex", "config", "model = \"gpt-5\"\n")
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Exists).To(BeTrue())

		saved, err := agentconfig.Read("codex", "config")
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.Content).To(Equal("model = \"gpt-5\"\n"))

		stat, err := os.Stat(filepath.Join(home, ".codex", "config.toml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(stat.Mode().Perm()).To(Equal(os.FileMode(0o600)))
	})

	It("rejects files outside the whitelist", func() {
		_, err := agentconfig.Read("claude", "arbitrary")
		Expect(err).To(MatchError(ContainSubstring("unknown agent config")))
		_, err = agentconfig.Write("codex", "../../etc/passwd", "x")
		Expect(err).To(MatchError(ContainSubstring("unknown agent config")))
	})
})

var _ = Describe("agent discovery and templates", func() {
	It("detects installed binaries with versions", func() {
		agents := agentconfig.DiscoverAgents()
		Expect(agents).To(HaveLen(3))
		slugs := []string{}
		for _, agent := range agents {
			slugs = append(slugs, agent.Slug)
			if agent.Installed {
				Expect(agent.Path).NotTo(BeEmpty())
			}
		}
		Expect(slugs).To(Equal([]string{"claude", "codex", "opencode"}))
	})

	It("provides a settings template for every agent", func() {
		for _, slug := range agentconfig.AgentOrder {
			templates := agentconfig.AgentTemplates[slug]
			Expect(templates).NotTo(BeEmpty(), "agent %s needs a template", slug)
			for _, template := range templates {
				// every template carries at least one placeholder to fill in
				Expect(template.Content).To(SatisfyAny(
					ContainSubstring("<YOUR_API_KEY>"),
					ContainSubstring("<MODEL>"),
					ContainSubstring("<PROVIDER_BASE_URL>"),
				))
				// every writable template maps onto the persistence whitelist
				if template.Name != "" {
					_, err := agentconfig.Read(slug, template.Name)
					Expect(err).NotTo(HaveOccurred(), "template %s/%s must be whitelisted", slug, template.Name)
				}
			}
		}
	})

	It("writes a template to disk through the whitelist", func() {
		home := GinkgoT().TempDir()
		original := os.Getenv("HOME")
		os.Setenv("HOME", home)
		DeferCleanup(func() { os.Setenv("HOME", original) })

		tpl := agentconfig.AgentTemplates["codex"][0]
		info, err := agentconfig.Write("codex", tpl.Name, tpl.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Content).To(ContainSubstring("<MODEL>"))
	})
})
