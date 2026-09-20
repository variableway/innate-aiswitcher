// Package market fetches the open models.dev catalog (a single api.json
// document listing every vendor and model with pricing and capabilities)
// and persists snapshots locally — the PocketBase SQLite database and/or a
// JSON file under ~/.innate-aiswitcher/market. The catalog is reference
// data: its models can be imported into vendor providers, where they share
// the provider's single API key. When the API is unreachable, reads keep
// serving from the local snapshot.
package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// DefaultBaseURL is the open models.dev catalog endpoint.
const DefaultBaseURL = "https://models.dev/api.json"

// LegacyLobehubURL is the retired lobehub tRPC source. Settings rows that
// still point at it are migrated to DefaultBaseURL on load.
const LegacyLobehubURL = "https://app.lobehub.com/trpc/lambda"

// Model mirrors one entry of the models.dev catalog: one vendor's model
// with its capabilities, context window and pricing.
type Model struct {
	ID                  string          `json:"id"`
	Identifier          string          `json:"identifier"`
	DisplayName         string          `json:"displayName"`
	Type                string          `json:"type,omitempty"`
	Category            string          `json:"category"`
	ProviderID          string          `json:"providerId"`
	Providers           []string        `json:"providers,omitempty"`
	ProviderCount       int             `json:"providerCount,omitempty"`
	ContextWindowTokens int             `json:"contextWindowTokens,omitempty"`
	Abilities           map[string]bool `json:"abilities,omitempty"`
	Pricing             json.RawMessage `json:"pricing,omitempty"`
	KnowledgeCutoff     string          `json:"knowledgeCutoff,omitempty"`
	Generation          string          `json:"generation,omitempty"`
	Family              string          `json:"family,omitempty"`
	Source              string          `json:"source,omitempty"`
	Enabled             *bool           `json:"enabled,omitempty"`
	ReleasedAt          string          `json:"releasedAt,omitempty"`
	Description         string          `json:"description,omitempty"`
}

// IsEnabled treats a missing enabled flag as enabled (only models.dev
// entries carrying a non-empty status — e.g. deprecated — are disabled).
func (m Model) IsEnabled() bool {
	return m.Enabled == nil || *m.Enabled
}

// Category is one vendor entry of the catalog's category list.
type Category struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// ModelsPage describes the fetched catalog. models.dev returns everything
// in one document, so a fetch is always a single page.
type ModelsPage struct {
	Items       []Model `json:"items"`
	CurrentPage int     `json:"currentPage"`
	PageSize    int     `json:"pageSize"`
	TotalCount  int     `json:"totalCount"`
	TotalPages  int     `json:"totalPages"`
}

// Client talks to a models.dev-compatible catalog API (one GET returning
// the whole JSON document). The zero value is not usable; use NewClient.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a client pointed at the public models.dev catalog.
func NewClient() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

// apiVendor is one vendor entry of the models.dev document; the map key is
// the vendor id and Name is the display name.
type apiVendor struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Models map[string]apiModel `json:"models"`
}

// apiModel is one model entry under a models.dev vendor. Field names mirror
// the upstream schema (see https://models.dev/api.json).
type apiModel struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Family           string          `json:"family"`
	Attachment       bool            `json:"attachment"`
	Reasoning        bool            `json:"reasoning"`
	ToolCall         bool            `json:"tool_call"`
	StructuredOutput bool            `json:"structured_output"`
	Modalities       *apiModalities  `json:"modalities"`
	OpenWeights      bool            `json:"open_weights"`
	Status           string          `json:"status"`
	Knowledge        string          `json:"knowledge"`
	ReleaseDate      string          `json:"release_date"`
	Limit            *apiLimit       `json:"limit"`
	Cost             json.RawMessage `json:"cost"`
}

// apiModalities is the per-direction modality split models.dev reports.
type apiModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type apiLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

// FetchAllModels downloads the whole catalog document and flattens it into
// one Model per vendor entry — vendors sorted by display name, each
// vendor's models sorted by id, so snapshots are deterministic.
func (c *Client) FetchAllModels(ctx context.Context) (*ModelsPage, []Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "innate-aiswitcher")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("market fetch: unexpected status %d", resp.StatusCode)
	}

	var vendors map[string]apiVendor
	if err := json.NewDecoder(resp.Body).Decode(&vendors); err != nil {
		return nil, nil, fmt.Errorf("market fetch: invalid catalog document: %w", err)
	}

	items := flattenCatalog(vendors)
	page := &ModelsPage{
		Items:       items,
		CurrentPage: 1,
		PageSize:    len(items),
		TotalCount:  len(items),
		TotalPages:  1,
	}
	return page, items, nil
}

// flattenCatalog maps the models.dev document onto catalog Model entries.
func flattenCatalog(vendors map[string]apiVendor) []Model {
	ordered := make([]apiVendor, 0, len(vendors))
	for id, vendor := range vendors {
		if vendor.ID == "" {
			vendor.ID = id
		}
		if len(vendor.Models) == 0 {
			continue
		}
		ordered = append(ordered, vendor)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Name != ordered[j].Name {
			return ordered[i].Name < ordered[j].Name
		}
		return ordered[i].ID < ordered[j].ID
	})

	items := make([]Model, 0, 512)
	for _, vendor := range ordered {
		category := vendor.Name
		if category == "" {
			category = vendor.ID
		}
		ids := make([]string, 0, len(vendor.Models))
		for id := range vendor.Models {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			entry := vendor.Models[id]
			items = append(items, catalogModel(vendor, id, entry))
		}
	}
	return items
}

func catalogModel(vendor apiVendor, id string, entry apiModel) Model {
	display := entry.Name
	if display == "" {
		display = id
	}
	abilities := map[string]bool{}
	if entry.ToolCall {
		abilities["tool_call"] = true
	}
	if entry.Reasoning {
		abilities["reasoning"] = true
	}
	if entry.StructuredOutput {
		abilities["structured_output"] = true
	}
	if entry.Attachment || acceptsImage(entry.Modalities) {
		abilities["multimodal"] = true
	}
	if entry.OpenWeights {
		abilities["open_weights"] = true
	}

	model := Model{
		ID:              vendor.ID + "/" + id,
		Identifier:      id,
		DisplayName:     display,
		Category:        vendor.Name,
		ProviderID:      vendor.ID,
		Providers:       []string{vendor.ID},
		ProviderCount:   1,
		Abilities:       abilities,
		KnowledgeCutoff: entry.Knowledge,
		Family:          entry.Family,
		ReleasedAt:      entry.ReleaseDate,
		Description:     entry.Description,
	}
	if model.Category == "" {
		model.Category = vendor.ID
	}
	if entry.Limit != nil {
		model.ContextWindowTokens = entry.Limit.Context
	}
	if len(entry.Cost) > 0 && string(entry.Cost) != "null" {
		model.Pricing = entry.Cost
	}
	if entry.Status != "" {
		disabled := false
		model.Enabled = &disabled
	}
	return model
}

// acceptsImage reports whether any modality direction handles images.
func acceptsImage(modalities *apiModalities) bool {
	if modalities == nil {
		return false
	}
	return hasModality(modalities.Input, "image") || hasModality(modalities.Output, "image")
}

func hasModality(modalities []string, want string) bool {
	for _, modality := range modalities {
		if strings.EqualFold(modality, want) {
			return true
		}
	}
	return false
}

// CategoriesFromModels derives the vendor category summary from a fetched
// model list (sorted by category).
func CategoriesFromModels(items []Model) []Category {
	counts := map[string]int{}
	for _, item := range items {
		if item.Category != "" {
			counts[item.Category]++
		}
	}
	categories := make([]Category, 0, len(counts))
	for name, count := range counts {
		categories = append(categories, Category{Category: name, Count: count})
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].Category < categories[j].Category })
	return categories
}
