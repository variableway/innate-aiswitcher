package market

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// catalogDoc is a miniature models.dev api.json document: two vendors with
// different shapes (full metadata vs minimal + deprecated status).
const catalogDoc = `{
  "minimax": {
    "id": "minimax", "name": "MiniMax",
    "models": {
      "MiniMax-M3": {
        "name": "MiniMax M3", "description": "flagship",
        "tool_call": true, "reasoning": true, "structured_output": false,
        "attachment": true, "modalities": {"input": ["text", "image"], "output": ["text"]},
        "knowledge": "2025-06", "release_date": "2025-09-01",
        "limit": {"context": 1000000, "output": 131072},
        "cost": {"input": 0.4, "output": 1.6, "cache_read": 0.04}
      }
    }
  },
  "zhipuai": {
    "id": "zhipuai", "name": "Zhipu AI",
    "models": {
      "glm-5.2": {"name": "GLM-5.2", "tool_call": true, "open_weights": true},
      "glm-4-old": {"name": "GLM 4 old", "status": "deprecated"}
    }
  }
}`

func serveCatalog(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(catalogDoc))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestFetchAllModelsFlattensCatalog(t *testing.T) {
	server := serveCatalog(t)
	client := NewClient()
	client.BaseURL = server.URL

	first, items, err := client.FetchAllModels(context.Background())
	if err != nil {
		t.Fatalf("FetchAllModels: %v", err)
	}
	if first.TotalCount != 3 || len(items) != 3 || first.TotalPages != 1 {
		t.Fatalf("expected all 3 models in one page, got first=%+v items=%d", first, len(items))
	}

	// Vendors sort by display name: MiniMax before Zhipu AI.
	if items[0].Category != "MiniMax" || items[0].Identifier != "MiniMax-M3" {
		t.Fatalf("first item must be MiniMax's model, got %+v", items[0])
	}
	m3 := items[0]
	if m3.ID != "minimax/MiniMax-M3" || m3.ProviderID != "minimax" {
		t.Fatalf("ids not mapped: %+v", m3)
	}
	if m3.DisplayName != "MiniMax M3" || m3.ContextWindowTokens != 1000000 {
		t.Fatalf("display/context not mapped: %+v", m3)
	}
	if !m3.Abilities["tool_call"] || !m3.Abilities["reasoning"] || !m3.Abilities["multimodal"] {
		t.Fatalf("abilities not mapped: %+v", m3.Abilities)
	}
	if m3.Abilities["structured_output"] {
		t.Fatalf("structured_output=false must stay absent: %+v", m3.Abilities)
	}
	var pricing struct {
		Input  float64 `json:"input"`
		Output float64 `json:"output"`
	}
	if err := json.Unmarshal(m3.Pricing, &pricing); err != nil || pricing.Input != 0.4 || pricing.Output != 1.6 {
		t.Fatalf("pricing not passed through: %s (%v)", m3.Pricing, err)
	}
	if m3.KnowledgeCutoff != "2025-06" || m3.ReleasedAt != "2025-09-01" || m3.Description != "flagship" {
		t.Fatalf("metadata not mapped: %+v", m3)
	}
	if !m3.IsEnabled() {
		t.Fatalf("model without status must be enabled: %+v", m3)
	}

	// glm-4-old carries a status and must be disabled; zhipuai models sort
	// by id within the vendor (glm-4-old before glm-5.2).
	old := items[1]
	if old.Identifier != "glm-4-old" || old.IsEnabled() {
		t.Fatalf("status-marked model must be disabled: %+v", old)
	}
	if items[2].Identifier != "glm-5.2" {
		t.Fatalf("models within a vendor must sort by id: %s, %s", items[1].Identifier, items[2].Identifier)
	}
}

func TestFetchAllModelsRejectsNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	if _, _, err := client.FetchAllModels(context.Background()); err == nil {
		t.Fatal("non-200 status must fail the fetch")
	} else if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error must mention the status: %v", err)
	}
}

func TestSettingsNormalization(t *testing.T) {
	if got := (Settings{}).Normalized().Storage; got != StorageSQLite {
		t.Fatalf("empty storage must default to sqlite, got %q", got)
	}
	if got := (Settings{Storage: "file"}).Normalized(); !got.UsesFile() || got.UsesSQLite() {
		t.Fatalf("file storage misconfigured: %+v", got)
	}
	both := Settings{Storage: "BOTH", SourceURL: ""}
	if !both.Normalized().UsesSQLite() || !both.Normalized().UsesFile() {
		t.Fatal("both storage must write to sqlite and file")
	}
	if got := both.Normalized().SourceURL; got != DefaultBaseURL {
		t.Fatalf("empty source must default to models.dev, got %q", got)
	}
	if got := (Settings{Storage: "nonsense"}).Normalized().Storage; got != StorageSQLite {
		t.Fatalf("invalid storage must fall back to sqlite, got %q", got)
	}
	if got := (Settings{SourceURL: LegacyLobehubURL}).Normalized().SourceURL; got != DefaultBaseURL {
		t.Fatalf("legacy lobehub source must migrate to models.dev, got %q", got)
	}
	if got := (Settings{SourceURL: "https://example.test/catalog.json"}).Normalized().SourceURL; got != "https://example.test/catalog.json" {
		t.Fatalf("custom source must be preserved, got %q", got)
	}
	if got := DefaultSettings(); !got.UsesSQLite() || !got.UsesFile() {
		t.Fatalf("default settings must back up to both backends: %+v", got)
	}
}

// testModel builds a minimal catalog entry for the persistence tests.
func testModel(identifier, category string) Model {
	enabled := true
	return Model{
		ID: category + "/" + identifier, Identifier: identifier, DisplayName: "Model " + identifier,
		Category: category, ProviderID: category, Providers: []string{category},
		ProviderCount: 1, ContextWindowTokens: 128000, Enabled: &enabled,
		Abilities: map[string]bool{"tool_call": true},
	}
}

func TestLoadCatalogFallsBackToFileSnapshot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AISW_MARKET_DIR", dir)

	snapshot := &Snapshot{
		Source: "test", Locale: "zh-CN", TotalCount: 1, ItemCount: 1, CategoryCount: 1,
		Items: []Model{
			{Identifier: "glm-5.2", DisplayName: "GLM 5.2", Category: "Zhipu AI", ProviderID: "zhipuai"},
			{Identifier: "MiniMax-M3", DisplayName: "MiniMax M3", Category: "MiniMax", ProviderID: "minimax"},
		},
	}
	if _, err := SaveFile(snapshot); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	catalog, err := LoadCatalog(&fakeCatalogStore{}, Filter{})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(catalog.Items) != 2 || catalog.Items[0].Identifier != "glm-5.2" {
		t.Fatalf("file snapshot must serve the read: %+v", catalog.Items)
	}
	if !catalog.HasData || catalog.Storage != StorageFile {
		t.Fatalf("fallback must mark hasData + file storage: %+v", catalog)
	}

	filtered, err := LoadCatalog(&fakeCatalogStore{}, Filter{Category: "MiniMax"})
	if err != nil {
		t.Fatalf("LoadCatalog filtered: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].Identifier != "MiniMax-M3" {
		t.Fatalf("category filter must apply to the fallback: %+v", filtered.Items)
	}
}

func TestLoadCatalogWithoutAnySnapshotIsEmpty(t *testing.T) {
	t.Setenv("AISW_MARKET_DIR", t.TempDir())

	catalog, err := LoadCatalog(&fakeCatalogStore{}, Filter{})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if catalog.HasData || len(catalog.Items) != 0 {
		t.Fatalf("no snapshot anywhere must be empty + hasData=false: %+v", catalog)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AISW_MARKET_DIR", dir)

	enabled := true
	snapshot := &Snapshot{
		Source: "test", Locale: "zh-CN", TotalCount: 1, ItemCount: 1, CategoryCount: 1,
		Items: []Model{{
			ID: "zhipuai/glm-5.2", Identifier: "glm-5.2", DisplayName: "GLM 5.2",
			Category: "Zhipu AI", ProviderID: "zhipuai", Enabled: &enabled,
		}},
	}
	path, err := SaveFile(snapshot)
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("snapshot written outside AISW_MARKET_DIR: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat snapshot: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("snapshot must be 0600, got %v", info.Mode().Perm())
	}

	loaded, err := LoadFile()
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded == nil || len(loaded.Items) != 1 || loaded.Items[0].Identifier != "glm-5.2" {
		t.Fatalf("snapshot did not round-trip: %+v", loaded)
	}
}

func TestLoadFileMissingReturnsNil(t *testing.T) {
	t.Setenv("AISW_MARKET_DIR", t.TempDir())
	loaded, err := LoadFile()
	if err != nil || loaded != nil {
		t.Fatalf("missing snapshot must return (nil, nil), got (%v, %v)", loaded, err)
	}
}

func TestModelIsEnabledDefaultsTrue(t *testing.T) {
	if !(Model{Identifier: "x"}).IsEnabled() {
		t.Fatal("missing enabled flag must default to enabled")
	}
	disabled := false
	if (Model{Identifier: "x", Enabled: &disabled}).IsEnabled() {
		t.Fatal("explicitly disabled model must report disabled")
	}
}

func TestCategoriesFromModelsSortsAndCounts(t *testing.T) {
	items := []Model{
		{Identifier: "a1", Category: "Zhipu AI"},
		{Identifier: "a2", Category: "Zhipu AI"},
		{Identifier: "b1", Category: "MiniMax"},
	}
	categories := CategoriesFromModels(items)
	if len(categories) != 2 || categories[0].Category != "MiniMax" || categories[0].Count != 1 {
		t.Fatalf("categories not sorted/counted: %+v", categories)
	}
	if categories[1].Category != "Zhipu AI" || categories[1].Count != 2 {
		t.Fatalf("categories not sorted/counted: %+v", categories)
	}
}
