// Package modelcatalog enriches locally configured models with community
// metadata (pricing per million tokens, multimodal flag, context window)
// from the open models.dev database — the same aggregated catalog that
// powers model-directory UIs like LobeHub's community pages. Vendor APIs
// do not expose this data, so it is fetched here on demand.
package modelcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

const (
	apiURL      = "https://models.dev/api.json"
	cacheTTL    = 5 * time.Minute
	httpTimeout = 20 * time.Second
)

// catalog is the subset of the models.dev schema we consume.
type catalog map[string]vendor

type vendor struct {
	Models map[string]modelEntry `json:"models"`
}

type modelEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Attachment bool   `json:"attachment"`
	ToolCall   bool   `json:"tool_call"`
	Reasoning  bool   `json:"reasoning"`
	Cost       *cost  `json:"cost"`
	Limit      *limit `json:"limit"`
}

type cost struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

type limit struct {
	Context int `json:"context"`
}

// vendorAliases maps aisw provider slugs to models.dev provider ids.
var vendorAliases = map[string][]string{
	"glm":       {"zhipuai", "volcengine"},
	"minimax":   {"minimax"},
	"kimi":      {"moonshotai", "moonshotai-cn"},
	"deepseek":  {"deepseek"},
	"openai":    {"openai"},
	"anthropic": {"anthropic"},
	"xiaomi":    {"xiaomi", "xiaomi-token-plan-cn"},
}

// VendorAliases lists the models.dev vendor ids an aisw provider slug maps
// to (e.g. glm → zhipuai). Shared with the market catalog so local lookups
// (offline model dropdowns) resolve the same vendors enrichment does.
func VendorAliases(slug string) []string {
	return vendorAliases[slug]
}

var (
	cacheMu   sync.Mutex
	cachedAt  time.Time
	cached    catalog
	fetchOnce sync.Once
)

// fetch retrieves the catalog (cached in-process for cacheTTL).
func fetch(ctx context.Context) (catalog, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cached != nil && time.Since(cachedAt) < cacheTTL {
		return cached, nil
	}
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model catalog unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model catalog returned status %d", resp.StatusCode)
	}
	var data catalog
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("model catalog decode: %w", err)
	}
	cached = data
	cachedAt = time.Now()
	return data, nil
}

// Result reports what happened during an enrichment run.
type Result struct {
	Matched int      `json:"matched"`
	Missed  []string `json:"missed,omitempty"`
	Updated int      `json:"updated"`
	Total   int      `json:"total"`
}

// Enrich fills missing model_meta fields (pricing, multimodal, note with
// context window / capabilities) for a provider's models from the catalog.
// Existing user-maintained values are never overwritten. Returns the
// updated provider (also when nothing changed) for the caller to persist.
func Enrich(ctx context.Context, provider store.Provider) (store.Provider, Result, error) {
	result := Result{Total: len(provider.Models), Missed: []string{}}
	if len(provider.Models) == 0 {
		return provider, result, nil
	}
	data, err := fetch(ctx)
	if err != nil {
		return provider, result, err
	}

	vendors := vendorAliases[provider.Slug]
	if len(vendors) == 0 {
		// fall back to the slug itself (custom providers may still match)
		vendors = []string{provider.Slug}
	}

	meta := map[string]store.ModelMeta{}
	for k, v := range provider.ModelMeta {
		meta[k] = v
	}

	for _, model := range provider.Models {
		entry, found := lookupModel(data, vendors, model)
		if !found {
			result.Missed = append(result.Missed, model)
			continue
		}
		result.Matched++

		existing := meta[model]
		before := existing
		if existing.InputPrice == "" && entry.Cost != nil && entry.Cost.Input > 0 {
			existing.InputPrice = fmt.Sprintf("$%g / 1M", entry.Cost.Input)
		}
		if existing.OutputPrice == "" && entry.Cost != nil && entry.Cost.Output > 0 {
			existing.OutputPrice = fmt.Sprintf("$%g / 1M", entry.Cost.Output)
		}
		if !existing.Multimodal && entry.Attachment {
			existing.Multimodal = true
		}
		if existing.Note == "" {
			var notes []string
			if entry.Limit != nil && entry.Limit.Context > 0 {
				notes = append(notes, fmt.Sprintf("ctx %s", humanCount(entry.Limit.Context)))
			}
			if entry.ToolCall {
				notes = append(notes, "tools")
			}
			if entry.Reasoning {
				notes = append(notes, "reasoning")
			}
			if entry.Name != "" && entry.Name != entry.ID {
				notes = append([]string{entry.Name}, notes...)
			}
			if len(notes) > 0 {
				existing.Note = strings.Join(notes, " · ")
			}
		}
		if existing != before {
			meta[model] = existing
			result.Updated++
		} else if _, ok := meta[model]; !ok {
			// keep an explicit empty marker out of the map when nothing applies
			continue
		}
	}

	if result.Updated > 0 {
		provider.ModelMeta = meta
	}
	if len(result.Missed) == 0 {
		result.Missed = nil
	}
	return provider, result, nil
}

// lookupModel finds a model entry by exact then fuzzy id/name match across
// the vendor candidates.
func lookupModel(data catalog, vendors []string, model string) (modelEntry, bool) {
	for _, vendorID := range vendors {
		v, ok := data[vendorID]
		if !ok {
			continue
		}
		if entry, ok := v.Models[model]; ok {
			return entry, true
		}
	}
	// fuzzy: case-insensitive containment either way
	lower := strings.ToLower(model)
	for _, vendorID := range vendors {
		v, ok := data[vendorID]
		if !ok {
			continue
		}
		for id, entry := range v.Models {
			idLower := strings.ToLower(id)
			if idLower == lower || strings.HasPrefix(idLower, lower) || strings.HasPrefix(lower, idLower) {
				return entry, true
			}
		}
	}
	return modelEntry{}, false
}

func humanCount(n int) string {
	switch {
	case n >= 1_000_000 && n%1_000_000 == 0:
		return fmt.Sprintf("%dM", n/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%dK", n/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
