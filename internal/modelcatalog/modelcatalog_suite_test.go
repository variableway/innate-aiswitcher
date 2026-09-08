package modelcatalog_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/modelcatalog"
	"github.com/variableway/innate-aiswitcher/internal/store"
)

func TestModelCatalogSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ModelCatalog Suite")
}

// The live network feed is exercised once; matching/mapping logic is what
// these specs pin down.
var _ = Describe("model catalog enrichment", func() {
	It("enriches a glm provider with pricing, modality and context notes", func() {
		provider := store.Provider{
			Slug: "glm", Models: []string{"glm-5.2", "glm-5.3"},
		}
		enriched, result, err := modelcatalog.Enrich(context.Background(), provider)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Matched).To(BeNumerically(">=", 1), "glm-5.2 must match zhipuai in the catalog")
		Expect(result.Total).To(Equal(2))

		meta := enriched.ModelMeta["glm-5.2"]
		Expect(meta.InputPrice).To(MatchRegexp(`\$\d`), "pricing filled, got %q", meta.InputPrice)
		Expect(meta.Note).To(ContainSubstring("ctx"))
	})

	It("fills minimax models and reports misses honestly", func() {
		provider := store.Provider{
			Slug:   "minimax",
			Models: []string{"MiniMax-M3", "definitely-not-a-model"},
		}
		enriched, result, err := modelcatalog.Enrich(context.Background(), provider)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Missed).To(ConsistOf("definitely-not-a-model"))
		if result.Matched > 0 {
			Expect(enriched.ModelMeta).To(HaveKey("MiniMax-M3"))
		}
	})

	It("never overwrites user-maintained metadata", func() {
		provider := store.Provider{
			Slug:   "glm",
			Models: []string{"glm-5.2"},
			ModelMeta: map[string]store.ModelMeta{
				"glm-5.2": {InputPrice: "MY-CUSTOM-PRICE"},
			},
		}
		enriched, _, err := modelcatalog.Enrich(context.Background(), provider)
		Expect(err).NotTo(HaveOccurred())
		Expect(enriched.ModelMeta["glm-5.2"].InputPrice).To(Equal("MY-CUSTOM-PRICE"))
	})

	It("maps kimi to the moonshot vendor ids", func() {
		provider := store.Provider{Slug: "kimi", Models: []string{"kimi-k2"}}
		_, result, err := modelcatalog.Enrich(context.Background(), provider)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Matched).To(BeNumerically(">=", 1))
	})
})
