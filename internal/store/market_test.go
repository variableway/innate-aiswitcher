package store

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/market"
)

func marketModel(identifier, category string) market.Model {
	enabled := true
	return market.Model{
		ID: identifier, Identifier: identifier, DisplayName: "Model " + identifier,
		Category: category, ProviderID: category, Providers: []string{category, "higress"},
		ProviderCount: 2, ContextWindowTokens: 131072, Enabled: &enabled,
		Abilities: map[string]bool{"vision": true},
	}
}

var _ = Describe("market catalog storage", func() {
	var s *Store

	BeforeEach(func() {
		s = newTestStore()
	})

	It("replaces the whole snapshot and restores every attribute", func() {
		Expect(s.ReplaceMarketModels([]market.Model{
			marketModel("glm-5.2", "zhipu"),
			marketModel("MiniMax-M3", "minimax"),
		})).To(Succeed())

		Expect(s.ReplaceMarketModels([]market.Model{marketModel("glm-5.3", "zhipu")})).To(Succeed())

		models, err := s.ListMarketModels(market.Filter{})
		Expect(err).NotTo(HaveOccurred())
		Expect(models).To(HaveLen(1), "replace must drop stale rows")
		Expect(models[0].Identifier).To(Equal("glm-5.3"))
		Expect(models[0].Providers).To(Equal([]string{"zhipu", "higress"}))
		Expect(models[0].ContextWindowTokens).To(Equal(131072))
		Expect(models[0].Abilities["vision"]).To(BeTrue())
	})

	It("filters by category and by identifier/display-name substring", func() {
		Expect(s.ReplaceMarketModels([]market.Model{
			marketModel("glm-5.2", "zhipu"),
			marketModel("MiniMax-M3", "minimax"),
			marketModel("glm-air", "zhipu"),
		})).To(Succeed())

		zhipu, err := s.ListMarketModels(market.Filter{Category: "zhipu"})
		Expect(err).NotTo(HaveOccurred())
		Expect(zhipu).To(HaveLen(2))

		byID, err := s.ListMarketModels(market.Filter{Query: "minimax-m"})
		Expect(err).NotTo(HaveOccurred())
		Expect(byID).To(HaveLen(1))
		Expect(byID[0].Identifier).To(Equal("MiniMax-M3"))

		byName, err := s.ListMarketModels(market.Filter{Query: "model glm-air"})
		Expect(err).NotTo(HaveOccurred())
		Expect(byName).To(HaveLen(1), "display_name must be searchable")
	})

	It("counts stored models", func() {
		Expect(s.MarketModelCount()).To(Equal(0))
		Expect(s.ReplaceMarketModels([]market.Model{marketModel("glm-5.2", "zhipu")})).To(Succeed())
		Expect(s.MarketModelCount()).To(Equal(1))
	})
})

var _ = Describe("settings KV", func() {
	var s *Store

	BeforeEach(func() {
		s = newTestStore()
	})

	It("round-trips JSON values", func() {
		var loaded struct {
			Storage   string    `json:"storage"`
			FetchedAt time.Time `json:"fetchedAt"`
		}
		ok, err := s.GetSetting("market", &loaded)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse(), "missing key must report false")

		Expect(s.SetSetting("market", map[string]any{"storage": "both"})).To(Succeed())
		ok, err = s.GetSetting("market", &loaded)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(loaded.Storage).To(Equal("both"))
	})

	It("overwrites the value on re-save", func() {
		Expect(s.SetSetting("market.meta", map[string]any{"itemCount": 1})).To(Succeed())
		Expect(s.SetSetting("market.meta", map[string]any{"itemCount": 2})).To(Succeed())

		var meta struct {
			ItemCount int `json:"itemCount"`
		}
		ok, err := s.GetSetting("market.meta", &meta)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(meta.ItemCount).To(Equal(2))
	})

	It("rejects an empty key", func() {
		Expect(s.SetSetting("", "x")).NotTo(Succeed())
	})
})

var _ = Describe("AddModels batch import", func() {
	var s *Store

	BeforeEach(func() {
		s = newTestStore()
	})

	It("appends many models in one upsert and shares the vendor key", func() {
		_, err := s.UpsertProvider(glmVendorRow())
		Expect(err).NotTo(HaveOccurred())

		saved, err := s.AddModels("glm", []string{"glm-5.3", "glm-5.4", " glm-5.2 "})
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.Models).To(Equal([]string{"glm-5.2", "glm-5.3", "glm-5.4"}), "existing and whitespace entries dedupe")
		Expect(saved.APIKey).To(Equal("ark-single-key"), "imported models share the vendor key")
		Expect(saved.DefaultModel).To(Equal("glm-5.2"), "default stays untouched")
	})

	It("is idempotent when every model already exists", func() {
		_, err := s.UpsertProvider(glmVendorRow())
		Expect(err).NotTo(HaveOccurred())

		saved, err := s.AddModels("glm", []string{"glm-5.2"})
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.Models).To(Equal([]string{"glm-5.2"}))
	})

	It("errors for an unknown provider", func() {
		_, err := s.AddModels("nope", []string{"m1"})
		Expect(err).To(HaveOccurred())
	})
})
