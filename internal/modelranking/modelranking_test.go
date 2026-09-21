package modelranking

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// apiDoc is a miniature models.dev api.json: a vendor with full metadata, a
// vendor without pricing, and a reseller serving another vendor's model
// (scores must attach to both).
const apiDoc = `{
  "zhipuai": {
    "id": "zhipuai", "name": "Zhipu AI",
    "models": {
      "glm-5.3": {
        "name": "GLM-5.3", "reasoning": true, "tool_call": true, "open_weights": true,
        "knowledge": "2026-07", "release_date": "2026-08-01",
        "limit": {"context": 256000, "output": 65536},
        "cost": {"input": 0.6, "output": 2.2}
      },
      "glm-4-old": {"name": "GLM 4 old", "tool_call": true}
    }
  },
  "moonshotai": {
    "id": "moonshotai", "name": "Moonshot AI",
    "models": {
      "kimi-k3": {
        "name": "Kimi K3", "reasoning": true,
        "release_date": "2026-07-15",
        "limit": {"context": 512000},
        "cost": {"input": 0.9, "output": 3.6}
      }
    }
  },
  "acme": {
    "id": "acme", "name": "Acme Relay",
    "models": {
      "glm-5.3": {
        "name": "GLM-5.3", "tool_call": true,
        "cost": {"input": 0.3, "output": 1.1}
      }
    }
  }
}`

// metaDoc is a miniature models.dev models.json with vendor-reported
// benchmarks. Suite names deliberately vary in spelling and apostrophes.
const metaDoc = `{
  "zhipuai/glm-5.3": {
    "id": "zhipuai/glm-5.3", "name": "GLM-5.3",
    "benchmarks": [
      {"name": "Terminal Bench 2.1", "score": 55.1, "metric": "success rate", "source": "https://zhipuai.example/glm53", "date": "2026-08-01"},
      {"name": "SWE-Bench Pro", "score": 63.5, "metric": "resolve rate", "source": "https://zhipuai.example/glm53", "date": "2026-08-01"},
      {"name": "Humanity's Last Exam", "score": 21.0, "metric": "accuracy", "source": "https://zhipuai.example/glm53", "date": "2026-08-01"},
      {"name": "SWE Bench Pro", "score": 61.2, "metric": "resolve rate", "source": "https://zhipuai.example/glm53-old", "date": "2026-06-01"}
    ]
  }
}`

// orDoc is a miniature OpenRouter /api/v1/models envelope. The :batch
// variant carries different (stale) scores and must lose to the base id;
// kimi-k3 has a null coding index, which OpenRouter omits entirely.
const orDoc = `{
  "data": [
    {"id": "z-ai/glm-5.3:batch", "benchmarks": {"artificial_analysis": {"intelligence_index": 99, "coding_index": 99, "agentic_index": 99}}},
    {"id": "z-ai/glm-5.3", "benchmarks": {"artificial_analysis": {"intelligence_index": 44.8, "coding_index": 74.8, "agentic_index": 51.2}}},
    {"id": "moonshotai/kimi-k3", "benchmarks": {"artificial_analysis": {"intelligence_index": 43.6, "agentic_index": 47.5}}},
    {"id": "openai/gpt-noindex", "benchmarks": {"design_arena": []}},
    {"id": "nobody/cares"}
  ]
}`

type feedConfig struct {
	apiStatus  int
	orStatus   int
	metaStatus int
}

// serveFeeds points the package feed URLs at httptest servers serving the
// fixture documents (or the given failure status) and drops the snapshot
// cache. Restores everything on test cleanup.
func serveFeeds(t *testing.T, cfg feedConfig) {
	t.Helper()
	serve := func(doc string, status int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if status != 0 && status != http.StatusOK {
				w.WriteHeader(status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(doc))
		}))
	}
	apiServer := serve(apiDoc, cfg.apiStatus)
	orServer := serve(orDoc, cfg.orStatus)
	metaServer := serve(metaDoc, cfg.metaStatus)

	prevAPI, prevOR, prevMeta := ModelsAPIURL, OpenRouterURL, ModelsMetaURL
	ModelsAPIURL, OpenRouterURL, ModelsMetaURL = apiServer.URL, orServer.URL, metaServer.URL
	resetCacheForTest()
	t.Cleanup(func() {
		ModelsAPIURL, OpenRouterURL, ModelsMetaURL = prevAPI, prevOR, prevMeta
		resetCacheForTest()
		apiServer.Close()
		orServer.Close()
		metaServer.Close()
	})
}

func findItem(t *testing.T, items []Item, provider, model string) Item {
	t.Helper()
	for _, item := range items {
		if item.ProviderID == provider && item.ModelID == model {
			return item
		}
	}
	t.Fatalf("item %s/%s not found in %d items", provider, model, len(items))
	return Item{}
}

func TestMetricCatalogIsComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, metric := range Metrics {
		if metric.ID == "" || metric.Name == "" || metric.Source == "" {
			t.Fatalf("metric %+v has empty required fields", metric)
		}
		if metric.Direction != directionAsc && metric.Direction != directionDesc {
			t.Fatalf("metric %s has invalid direction %q", metric.ID, metric.Direction)
		}
		if seen[metric.ID] {
			t.Fatalf("duplicate metric id %s", metric.ID)
		}
		seen[metric.ID] = true
	}
	if len(Metrics) != 8 {
		t.Fatalf("expected 8 metrics, got %d", len(Metrics))
	}
	if _, ok := MetricByID(MetricCoding); !ok {
		t.Fatal("default metric coding missing from catalog")
	}
	if _, ok := MetricByID("nope"); ok {
		t.Fatal("unknown metric id must not resolve")
	}
}

func TestLoadMergesScoresAndBenchmarks(t *testing.T) {
	serveFeeds(t, feedConfig{})
	result, err := Load(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Metric != MetricCoding {
		t.Fatalf("default metric must be coding, got %s", result.Metric)
	}
	if result.Total != 4 || len(result.Items) != 4 {
		t.Fatalf("expected 4 items, got total=%d items=%d", result.Total, len(result.Items))
	}

	// Scores attach by model id across every provider serving the model.
	for _, provider := range []string{"zhipuai", "acme"} {
		item := findItem(t, result.Items, provider, "glm-5.3")
		if item.Coding == nil || *item.Coding != 74.8 {
			t.Fatalf("%s/glm-5.3 coding score not attached: %+v", provider, item.Coding)
		}
		if item.Intelligence == nil || *item.Intelligence != 44.8 {
			t.Fatalf("%s/glm-5.3 intelligence not attached", provider)
		}
		if item.Agentic == nil || *item.Agentic != 51.2 {
			t.Fatalf("%s/glm-5.3 agentic not attached", provider)
		}
	}

	// A null AA index stays absent while the others survive.
	kimi := findItem(t, result.Items, "moonshotai", "kimi-k3")
	if kimi.Coding != nil {
		t.Fatalf("kimi-k3 coding must be nil, got %v", *kimi.Coding)
	}
	if kimi.Intelligence == nil || *kimi.Intelligence != 43.6 {
		t.Fatalf("kimi-k3 intelligence not attached: %+v", kimi.Intelligence)
	}

	// Vendor-reported benchmarks join by vendor/model key on the vendor's
	// own entry and fall back to model id for resellers.
	zhipu := findItem(t, result.Items, "zhipuai", "glm-5.3")
	if len(zhipu.Benchmarks) != 4 {
		t.Fatalf("expected 4 benchmarks on zhipuai/glm-5.3, got %d", len(zhipu.Benchmarks))
	}
	// Normalized names sort the two SWE-Bench Pro spellings adjacent.
	adjacent := false
	for i := 0; i+1 < len(zhipu.Benchmarks); i++ {
		if normalizeBenchName(zhipu.Benchmarks[i].Name) == normalizeBenchName(zhipu.Benchmarks[i+1].Name) {
			adjacent = true
		}
	}
	if !adjacent {
		t.Fatalf("SWE-Bench Pro variants must sort adjacent: %+v", zhipu.Benchmarks)
	}
	acme := findItem(t, result.Items, "acme", "glm-5.3")
	if len(acme.Benchmarks) != 4 {
		t.Fatalf("reseller must inherit benchmarks by model id, got %d", len(acme.Benchmarks))
	}

	// Catalog facts map through: pricing, context, capability flags.
	if zhipu.InputPrice == nil || *zhipu.InputPrice != 0.6 || zhipu.OutputPrice == nil || *zhipu.OutputPrice != 2.2 {
		t.Fatalf("zhipuai pricing not mapped: %+v %+v", zhipu.InputPrice, zhipu.OutputPrice)
	}
	if zhipu.Context != 256000 || zhipu.OutputLimit != 65536 {
		t.Fatalf("zhipuai limits not mapped: %+v", zhipu)
	}
	if !zhipu.OpenWeights || !zhipu.Reasoning || !zhipu.ToolCall {
		t.Fatalf("zhipuai flags not mapped: %+v", zhipu)
	}
	if zhipu.Knowledge != "2026-07" || zhipu.ReleaseDate != "2026-08-01" {
		t.Fatalf("zhipuai dates not mapped: %+v", zhipu)
	}
}

func TestLoadDedupesOpenRouterVariants(t *testing.T) {
	serveFeeds(t, feedConfig{})
	result, err := Load(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	item := findItem(t, result.Items, "zhipuai", "glm-5.3")
	// The :batch variant was served first with different scores; the exact
	// base id must win.
	if item.Coding == nil || *item.Coding != 74.8 || *item.Intelligence != 44.8 {
		t.Fatalf("variant scores leaked: coding=%v intelligence=%v", item.Coding, item.Intelligence)
	}
}

func TestLoadSortsByMetricAndSinksMissing(t *testing.T) {
	serveFeeds(t, feedConfig{})

	// price_input asc: acme 0.3 < zhipuai 0.6 < moonshot 0.9, then glm-4-old (no price).
	result, err := Load(context.Background(), Filter{Metric: MetricPriceInput})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantOrder := []string{"acme/glm-5.3", "zhipuai/glm-5.3", "moonshotai/kimi-k3", "zhipuai/glm-4-old"}
	for i, want := range wantOrder {
		got := result.Items[i].ProviderID + "/" + result.Items[i].ModelID
		if got != want {
			t.Fatalf("price_input rank %d: want %s, got %s", i+1, want, got)
		}
	}

	// coding desc: glm-5.3 entries (74.8) first, kimi (nil) and glm-4-old sink.
	result, err = Load(context.Background(), Filter{Metric: MetricCoding})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	first := result.Items[0]
	if first.ModelID != "glm-5.3" || first.Coding == nil {
		t.Fatalf("coding top rank must be a glm-5.3 entry, got %+v", first)
	}
	last := result.Items[len(result.Items)-1]
	if last.Coding != nil {
		t.Fatalf("entries without the metric must sink to the bottom, got %+v", last)
	}

	// context desc: 512K kimi first, then 256K glm, then entries without a limit.
	result, err = Load(context.Background(), Filter{Metric: MetricContext})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Items[0].ModelID != "kimi-k3" {
		t.Fatalf("context top rank must be kimi-k3, got %+v", result.Items[0])
	}

	// released desc: glm-5.3 (2026-08-01) before kimi-k3 (2026-07-15).
	result, err = Load(context.Background(), Filter{Metric: MetricReleased})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Items[0].ModelID != "glm-5.3" || result.Items[1].ModelID != "kimi-k3" {
		t.Fatalf("released order wrong: %s then %s", result.Items[0].ModelID, result.Items[1].ModelID)
	}
}

func TestLoadRejectsUnknownMetric(t *testing.T) {
	serveFeeds(t, feedConfig{})
	if _, err := Load(context.Background(), Filter{Metric: "charm"}); err == nil {
		t.Fatal("unknown metric must fail")
	}
}

func TestLoadQueryFilter(t *testing.T) {
	serveFeeds(t, feedConfig{})

	result, err := Load(context.Background(), Filter{Query: "GLM"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Total != 3 { // zhipuai glm-5.3, zhipuai glm-4-old, acme glm-5.3
		t.Fatalf("query glm: want 3, got %d", result.Total)
	}

	result, err = Load(context.Background(), Filter{Query: "moonshot"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Total != 1 || result.Items[0].ModelID != "kimi-k3" {
		t.Fatalf("query moonshot: want kimi-k3 only, got %d", result.Total)
	}
}

func TestLoadLimitAndTotal(t *testing.T) {
	serveFeeds(t, feedConfig{})
	result, err := Load(context.Background(), Filter{Limit: 2})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(result.Items) != 2 || result.Total != 4 {
		t.Fatalf("limit: want 2 of 4, got %d of %d", len(result.Items), result.Total)
	}
}

func TestLoadOpenRouterDownDegradesGracefully(t *testing.T) {
	serveFeeds(t, feedConfig{orStatus: http.StatusInternalServerError})
	result, err := Load(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("OpenRouter outage must not fail the request: %v", err)
	}
	var orSource *SourceStatus
	for i := range result.Sources {
		if result.Sources[i].ID == SourceOpenRouter {
			orSource = &result.Sources[i]
		}
	}
	if orSource == nil || orSource.OK {
		t.Fatalf("openrouter source must be reported unhealthy, got %+v", result.Sources)
	}
	item := findItem(t, result.Items, "zhipuai", "glm-5.3")
	if item.Coding != nil || item.Intelligence != nil {
		t.Fatalf("scores must be absent when the feed is down: %+v", item)
	}
	if len(item.Benchmarks) == 0 {
		t.Fatal("models.json benchmarks must survive an OpenRouter outage")
	}
}

func TestLoadCatalogDownFailsRequest(t *testing.T) {
	serveFeeds(t, feedConfig{apiStatus: http.StatusServiceUnavailable})
	if _, err := Load(context.Background(), Filter{}); err == nil {
		t.Fatal("models.dev catalog outage must fail the request")
	}
}

func TestBaseModelIDAndNormalize(t *testing.T) {
	cases := map[string]string{"z-ai/glm-5.3": "z-ai/glm-5.3", "z-ai/glm-5.3:batch": "z-ai/glm-5.3", "x:free": "x", "y": "y"}
	for in, want := range cases {
		if got := baseModelID(in); got != want {
			t.Fatalf("baseModelID(%q) = %q, want %q", in, got, want)
		}
	}
	if normalizeBenchName("SWE-Bench Pro") != normalizeBenchName("SWE Bench Pro") ||
		normalizeBenchName("Humanity's Last Exam") != normalizeBenchName("Humanity’s Last Exam") {
		t.Fatal("benchmark name normalization must unify spelling variants")
	}
	keyCases := map[string]string{
		"z-ai/glm-5.3":       "glm-5.3",
		"z-ai/glm-5.3:batch": "glm-5.3",
		"moonshotai/kimi-k3": "kimi-k3",
		"bare-model":         "bare-model",
		"a/b/c-deep":         "c-deep",
	}
	for in, want := range keyCases {
		if got := orModelKey(in); got != want {
			t.Fatalf("orModelKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestLoadResponseShapeJSON guards the wire contract the web page relies on.
func TestLoadResponseShapeJSON(t *testing.T) {
	serveFeeds(t, feedConfig{})
	result, err := Load(context.Background(), Filter{Metric: MetricIntelligence, Limit: 1})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"metrics", "metric", "items", "total", "fetchedAt", "sources"} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("response missing key %q", key)
		}
	}
}
