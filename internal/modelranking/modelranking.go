// Package modelranking merges the open models.dev catalog with independent
// Artificial Analysis scores (redistributed by OpenRouter's public models
// API) and vendor-reported benchmarks (models.dev models.json) into one
// sortable model list. Scores describe the model itself, so they attach by
// model id across every provider serving that model; prices and context
// windows stay per provider entry. Nothing is persisted — the merged
// snapshot is cached in-process for an hour. Vendor-reported benchmarks are
// display-only: harnesses and prompts differ per vendor, so they never
// drive ranking order.
package modelranking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrUnknownMetric reports a filter naming a metric outside the catalog;
// handlers map it to a client error.
var ErrUnknownMetric = errors.New("unknown metric")

// Upstream feed endpoints; fields are plain vars so tests can point them at
// httptest servers.
var (
	ModelsAPIURL  = "https://models.dev/api.json"
	ModelsMetaURL = "https://models.dev/models.json"
	OpenRouterURL = "https://openrouter.ai/api/v1/models"
)

const (
	httpTimeout = 60 * time.Second
	snapshotTTL = time.Hour
	defaultRank = 100
	maxRank     = 500
)

// Source identifiers reported in Result.Sources.
const (
	SourceModelsAPI    = "models_dev"
	SourceModelsMeta   = "models_dev_meta"
	SourceOpenRouter   = "openrouter"
	SourceArtificialAA = "artificial_analysis"
	MetricIntelligence = "intelligence"
	MetricCoding       = "coding"
	MetricAgentic      = "agentic"
	MetricPriceInput   = "price_input"
	MetricPriceOutput  = "price_output"
	MetricContext      = "context"
	MetricOutputLimit  = "output_limit"
	MetricReleased     = "released"
	directionDesc      = "desc"
	directionAsc       = "asc"
	unitIndex          = "index"
	unitUSDPerM        = "usd_per_1m"
	unitTokens         = "tokens"
	unitDate           = "date"
)

// Metric describes one rankable indicator: where its value comes from and
// which direction is better. The full catalog is returned with every result
// so clients can render a metric picker without a second request.
type Metric struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Direction   string `json:"direction"` // "desc" (higher/newer is better) or "asc" (lower is better)
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description"`
}

// Metrics is the catalog of every rankable metric. Intelligence/coding/
// agentic come from Artificial Analysis (independent, unified evaluation);
// the rest come from the models.dev directory itself.
var Metrics = []Metric{
	{ID: MetricCoding, Name: "AA Coding Index", Source: SourceArtificialAA, Direction: directionDesc, Unit: unitIndex,
		Description: "Independent coding-ability index by Artificial Analysis (via OpenRouter)."},
	{ID: MetricIntelligence, Name: "AA Intelligence Index", Source: SourceArtificialAA, Direction: directionDesc, Unit: unitIndex,
		Description: "Independent overall-intelligence index by Artificial Analysis (via OpenRouter)."},
	{ID: MetricAgentic, Name: "AA Agentic Index", Source: SourceArtificialAA, Direction: directionDesc, Unit: unitIndex,
		Description: "Independent tool-use/agentic index by Artificial Analysis (via OpenRouter)."},
	{ID: MetricPriceInput, Name: "Input price", Source: SourceModelsAPI, Direction: directionAsc, Unit: unitUSDPerM,
		Description: "Listed input price per 1M tokens (models.dev directory)."},
	{ID: MetricPriceOutput, Name: "Output price", Source: SourceModelsAPI, Direction: directionAsc, Unit: unitUSDPerM,
		Description: "Listed output price per 1M tokens (models.dev directory)."},
	{ID: MetricContext, Name: "Context window", Source: SourceModelsAPI, Direction: directionDesc, Unit: unitTokens,
		Description: "Context window size in tokens (models.dev directory)."},
	{ID: MetricOutputLimit, Name: "Max output", Source: SourceModelsAPI, Direction: directionDesc, Unit: unitTokens,
		Description: "Maximum output tokens (models.dev directory)."},
	{ID: MetricReleased, Name: "Release date", Source: SourceModelsAPI, Direction: directionDesc, Unit: unitDate,
		Description: "Model release date, newest first (models.dev directory)."},
}

// MetricByID looks a metric up in the catalog.
func MetricByID(id string) (Metric, bool) {
	for _, metric := range Metrics {
		if metric.ID == id {
			return metric, true
		}
	}
	return Metric{}, false
}

// VendorBenchmark is one vendor-reported benchmark score from models.dev
// models.json. The source links to the vendor's own announcement, so these
// are claims, not independent measurements.
type VendorBenchmark struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Metric  string  `json:"metric,omitempty"`
	Source  string  `json:"source,omitempty"`
	Date    string  `json:"date,omitempty"`
	Harness string  `json:"harness,omitempty"`
	Version string  `json:"version,omitempty"`
}

// Item is one provider-served model with every metric attached. Score
// fields are nil when the upstream has no value.
type Item struct {
	ProviderID   string            `json:"providerId"`
	ProviderName string            `json:"providerName"`
	ModelID      string            `json:"modelId"`
	Name         string            `json:"name"`
	Intelligence *float64          `json:"intelligence,omitempty"`
	Coding       *float64          `json:"coding,omitempty"`
	Agentic      *float64          `json:"agentic,omitempty"`
	InputPrice   *float64          `json:"inputPrice,omitempty"`
	OutputPrice  *float64          `json:"outputPrice,omitempty"`
	Context      int               `json:"context,omitempty"`
	OutputLimit  int               `json:"outputLimit,omitempty"`
	OpenWeights  bool              `json:"openWeights"`
	Reasoning    bool              `json:"reasoning"`
	ToolCall     bool              `json:"toolCall"`
	Attachment   bool              `json:"attachment"`
	Knowledge    string            `json:"knowledge,omitempty"`
	ReleaseDate  string            `json:"releaseDate,omitempty"`
	Benchmarks   []VendorBenchmark `json:"benchmarks,omitempty"`
}

// SourceStatus reports the health of one upstream feed so the UI can flag
// partial data instead of silently showing empty scores.
type SourceStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url,omitempty"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// Snapshot is one merged view of all feeds.
type Snapshot struct {
	Items     []Item
	Sources   []SourceStatus
	FetchedAt time.Time
}

// Filter narrows and orders a Result.
type Filter struct {
	Metric string // metric id; empty defaults to coding
	Query  string // case-insensitive substring over provider/model names
	Limit  int    // 0 → defaultRank, capped at maxRank
}

// Result is the API payload for GET /api/aisw/rankings.
type Result struct {
	Metrics   []Metric       `json:"metrics"`
	Metric    string         `json:"metric"`
	Items     []Item         `json:"items"`
	Total     int            `json:"total"`
	FetchedAt time.Time      `json:"fetchedAt"`
	Sources   []SourceStatus `json:"sources"`
}

// apiVendor is one vendor entry of the models.dev api.json document.
type apiVendor struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Models map[string]apiModel `json:"models"`
}

// apiModel is one model entry under a models.dev vendor.
type apiModel struct {
	Name        string    `json:"name"`
	Attachment  bool      `json:"attachment"`
	Reasoning   bool      `json:"reasoning"`
	ToolCall    bool      `json:"tool_call"`
	OpenWeights bool      `json:"open_weights"`
	Knowledge   string    `json:"knowledge"`
	ReleaseDate string    `json:"release_date"`
	Limit       *apiLimit `json:"limit"`
	Cost        *apiCost  `json:"cost"`
}

type apiLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

type apiCost struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

// metaModel is the subset of models.dev models.json entries we consume:
// provider-agnostic model facts plus vendor-reported benchmarks.
type metaModel struct {
	Benchmarks []VendorBenchmark `json:"benchmarks"`
}

// orModels is the OpenRouter /api/v1/models envelope.
type orModels struct {
	Data []orModel `json:"data"`
}

// orModel is one OpenRouter model entry; only the Artificial Analysis
// indexes are consumed. Indexes are pointers: OpenRouter omits a model's
// null indexes from the payload entirely.
type orModel struct {
	ID         string `json:"id"`
	Benchmarks *struct {
		ArtificialAnalysis *struct {
			IntelligenceIndex *float64 `json:"intelligence_index"`
			CodingIndex       *float64 `json:"coding_index"`
			AgenticIndex      *float64 `json:"agentic_index"`
		} `json:"artificial_analysis"`
	} `json:"benchmarks"`
}

// aaScores carries the three Artificial Analysis indexes for one model.
type aaScores struct {
	Intelligence *float64
	Coding       *float64
	Agentic      *float64
}

var (
	cacheMu   sync.Mutex
	cached    *Snapshot
	cachedAt  time.Time
	httpFetch = &http.Client{Timeout: httpTimeout}
)

// Load returns the merged ranking view for one filter, refreshing the
// cached snapshot when stale. A failure of the models.dev catalog (the base
// list) fails the call; failures of the enrichment feeds (OpenRouter,
// models.json) only mark the corresponding source unhealthy.
func Load(ctx context.Context, filter Filter) (Result, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cached == nil || time.Since(cachedAt) >= snapshotTTL {
		snap, err := buildSnapshot(ctx)
		if err != nil {
			return Result{}, err
		}
		cached = snap
		cachedAt = time.Now()
	}
	return cached.select_(filter)
}

// resetCacheForTest drops the cached snapshot (tests point the feed URLs at
// per-test httptest servers).
func resetCacheForTest() {
	cached = nil
}

func buildSnapshot(ctx context.Context) (*Snapshot, error) {
	snap := &Snapshot{FetchedAt: time.Now(), Sources: []SourceStatus{}}

	var vendors map[string]apiVendor
	if err := fetchJSON(ctx, ModelsAPIURL, &vendors); err != nil {
		snap.Sources = append(snap.Sources, SourceStatus{ID: SourceModelsAPI, Name: "models.dev catalog", URL: ModelsAPIURL, OK: false, Detail: err.Error()})
		return nil, fmt.Errorf("model ranking: catalog feed unavailable: %w", err)
	}
	snap.Sources = append(snap.Sources, SourceStatus{ID: SourceModelsAPI, Name: "models.dev catalog", URL: ModelsAPIURL, OK: true})

	scores := map[string]*aaScores{}
	var orResp orModels
	if err := fetchJSON(ctx, OpenRouterURL, &orResp); err != nil {
		snap.Sources = append(snap.Sources, SourceStatus{ID: SourceOpenRouter, Name: "OpenRouter (Artificial Analysis scores)", URL: OpenRouterURL, OK: false, Detail: err.Error()})
	} else {
		snap.Sources = append(snap.Sources, SourceStatus{ID: SourceOpenRouter, Name: "OpenRouter (Artificial Analysis scores)", URL: OpenRouterURL, OK: true})
		scores = scoreIndex(orResp.Data)
	}

	meta := map[string]metaModel{}
	if err := fetchJSON(ctx, ModelsMetaURL, &meta); err != nil {
		snap.Sources = append(snap.Sources, SourceStatus{ID: SourceModelsMeta, Name: "models.dev model metadata", URL: ModelsMetaURL, OK: false, Detail: err.Error()})
	} else {
		snap.Sources = append(snap.Sources, SourceStatus{ID: SourceModelsMeta, Name: "models.dev model metadata", URL: ModelsMetaURL, OK: true})
	}

	snap.Items = mergeItems(vendors, scores, meta)
	return snap, nil
}

// fetchJSON GETs a JSON document into dst.
func fetchJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "innate-aiswitcher")
	resp, err := httpFetch.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("invalid document: %w", err)
	}
	return nil
}

// scoreIndex deduplicates OpenRouter model variants (:batch, :free, …)
// down to one entry per base model id, keyed by lowercase model id so the
// same score attaches to every provider serving the model.
func scoreIndex(models []orModel) map[string]*aaScores {
	index := map[string]*aaScores{}
	for _, model := range models {
		if model.Benchmarks == nil || model.Benchmarks.ArtificialAnalysis == nil {
			continue
		}
		aa := model.Benchmarks.ArtificialAnalysis
		if aa.IntelligenceIndex == nil && aa.CodingIndex == nil && aa.AgenticIndex == nil {
			continue
		}
		key := orModelKey(model.ID)
		if key == "" {
			continue
		}
		score := &aaScores{Intelligence: aa.IntelligenceIndex, Coding: aa.CodingIndex, Agentic: aa.AgenticIndex}
		if existing, exists := index[key]; exists {
			if model.ID == baseModelID(model.ID) {
				*existing = *score // a variant-less id wins over a ":variant" seen first
			}
			continue
		}
		index[key] = score
	}
	return index
}

// baseModelID strips OpenRouter variant suffixes like ":batch" or ":free".
func baseModelID(id string) string {
	if i := strings.Index(id, ":"); i >= 0 {
		return id[:i]
	}
	return id
}

// orModelKey normalizes an OpenRouter model id ("z-ai/glm-5.3:batch") down
// to the bare lowercase model id ("glm-5.3") so scores join models.dev
// entries by model id regardless of vendor namespace.
func orModelKey(id string) string {
	base := baseModelID(id)
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	return strings.ToLower(base)
}

// mergeItems flattens the catalog and attaches scores (by model id) and
// vendor-reported benchmarks (by "vendor/model" key, falling back to model
// id). Vendors and models iterate in name order for deterministic output.
func mergeItems(vendors map[string]apiVendor, scores map[string]*aaScores, meta map[string]metaModel) []Item {
	vendorIDs := make([]string, 0, len(vendors))
	for id := range vendors {
		vendorIDs = append(vendorIDs, id)
	}
	sort.Slice(vendorIDs, func(i, j int) bool {
		vi, vj := vendors[vendorIDs[i]], vendors[vendorIDs[j]]
		if vi.Name != vj.Name {
			return vi.Name < vj.Name
		}
		return vendorIDs[i] < vendorIDs[j]
	})

	// model-id → benchmarks index: models.json entries are keyed by their
	// canonical vendor, but the benchmarks describe the model itself, so any
	// provider serving the same model id inherits them.
	byModel := map[string][]VendorBenchmark{}
	for key, m := range meta {
		if len(m.Benchmarks) == 0 {
			continue
		}
		modelID := key
		if i := strings.LastIndex(key, "/"); i >= 0 {
			modelID = key[i+1:]
		}
		if _, exists := byModel[strings.ToLower(modelID)]; !exists {
			byModel[strings.ToLower(modelID)] = sortBenchmarks(m.Benchmarks)
		}
	}

	items := make([]Item, 0, 512)
	for _, vendorID := range vendorIDs {
		vendor := vendors[vendorID]
		name := vendor.Name
		if name == "" {
			name = vendorID
		}
		modelIDs := make([]string, 0, len(vendor.Models))
		for id := range vendor.Models {
			modelIDs = append(modelIDs, id)
		}
		sort.Strings(modelIDs)
		for _, modelID := range modelIDs {
			entry := vendor.Models[modelID]
			display := entry.Name
			if display == "" {
				display = modelID
			}
			item := Item{
				ProviderID:   vendorID,
				ProviderName: name,
				ModelID:      modelID,
				Name:         display,
				OpenWeights:  entry.OpenWeights,
				Reasoning:    entry.Reasoning,
				ToolCall:     entry.ToolCall,
				Attachment:   entry.Attachment,
				Knowledge:    entry.Knowledge,
				ReleaseDate:  entry.ReleaseDate,
			}
			if entry.Limit != nil {
				item.Context = entry.Limit.Context
				item.OutputLimit = entry.Limit.Output
			}
			if entry.Cost != nil {
				input, output := entry.Cost.Input, entry.Cost.Output
				if input > 0 {
					item.InputPrice = &input
				}
				if output > 0 {
					item.OutputPrice = &output
				}
			}
			if score, ok := scores[strings.ToLower(modelID)]; ok {
				item.Intelligence = score.Intelligence
				item.Coding = score.Coding
				item.Agentic = score.Agentic
			}
			if m, ok := meta[vendorID+"/"+modelID]; ok && len(m.Benchmarks) > 0 {
				item.Benchmarks = sortBenchmarks(m.Benchmarks)
			} else if m, ok := byModel[strings.ToLower(modelID)]; ok {
				item.Benchmarks = m
			}
			items = append(items, item)
		}
	}
	return items
}

// sortBenchmarks orders vendor-reported benchmarks by normalized name so
// spelling variants of the same suite stay adjacent.
func sortBenchmarks(benchmarks []VendorBenchmark) []VendorBenchmark {
	sorted := append([]VendorBenchmark(nil), benchmarks...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return normalizeBenchName(sorted[i].Name) < normalizeBenchName(sorted[j].Name)
	})
	return sorted
}

// normalizeBenchName casefolds a suite name and strips separators and
// typographic apostrophes so "SWE-Bench Pro", "SWE Bench Pro" and
// "SWE-bench Pro" compare equal.
func normalizeBenchName(name string) string {
	repl := strings.NewReplacer(" ", "", "-", "", "'", "", "’", "", "‘", "")
	return repl.Replace(strings.ToLower(name))
}

// select_ applies the filter to a snapshot: metric validation, substring
// query, ordering (entries missing the metric value sink to the bottom) and
// the row limit.
func (s *Snapshot) select_(filter Filter) (Result, error) {
	metric, ok := MetricByID(filter.Metric)
	if !ok {
		if filter.Metric == "" {
			metric, _ = MetricByID(MetricCoding)
		} else {
			return Result{}, fmt.Errorf("%w %q (see metrics catalog)", ErrUnknownMetric, filter.Metric)
		}
	}

	query := strings.ToLower(strings.TrimSpace(filter.Query))
	items := make([]Item, 0, len(s.Items))
	for _, item := range s.Items {
		if query != "" &&
			!strings.Contains(strings.ToLower(item.ProviderID+" "+item.ProviderName+" "+item.ModelID+" "+item.Name), query) {
			continue
		}
		items = append(items, item)
	}
	total := len(items)

	sort.SliceStable(items, func(i, j int) bool {
		vi, oki := metricValue(items[i], metric.ID)
		vj, okj := metricValue(items[j], metric.ID)
		if oki != okj {
			return oki // entries with a value rank above entries without
		}
		if oki && vi != vj {
			if metric.Direction == directionAsc {
				return vi < vj
			}
			return vi > vj
		}
		if items[i].ProviderName != items[j].ProviderName {
			return items[i].ProviderName < items[j].ProviderName
		}
		return items[i].ModelID < items[j].ModelID
	})

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultRank
	}
	if limit > maxRank {
		limit = maxRank
	}
	if len(items) > limit {
		items = items[:limit]
	}

	return Result{
		Metrics:   Metrics,
		Metric:    metric.ID,
		Items:     items,
		Total:     total,
		FetchedAt: s.FetchedAt,
		Sources:   s.Sources,
	}, nil
}

// metricValue extracts one item's comparable value for a metric id; ok is
// false when the item has no value for it.
func metricValue(item Item, metricID string) (value float64, ok bool) {
	switch metricID {
	case MetricIntelligence:
		return deref(item.Intelligence)
	case MetricCoding:
		return deref(item.Coding)
	case MetricAgentic:
		return deref(item.Agentic)
	case MetricPriceInput:
		return deref(item.InputPrice)
	case MetricPriceOutput:
		return deref(item.OutputPrice)
	case MetricContext:
		return float64(item.Context), item.Context > 0
	case MetricOutputLimit:
		return float64(item.OutputLimit), item.OutputLimit > 0
	case MetricReleased:
		if t, err := time.Parse("2006-01-02", item.ReleaseDate); err == nil {
			return float64(t.Unix()), true
		}
		return 0, false
	}
	return 0, false
}

func deref(v *float64) (float64, bool) {
	if v == nil {
		return 0, false
	}
	return *v, true
}
