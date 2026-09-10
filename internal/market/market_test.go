package market

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// fakeMarket serves lobehub-shaped tRPC batch envelopes for the client tests.
type fakeMarket struct {
	pages    map[int][]Model
	requests []map[string]any
}

func (f *fakeMarket) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input map[string]any
		if err := json.Unmarshal([]byte(r.URL.Query().Get("input")), &input); err != nil {
			t.Errorf("invalid input param: %v", err)
			http.Error(w, "bad input", http.StatusBadRequest)
			return
		}
		f.requests = append(f.requests, input)

		zero := input["0"].(map[string]any)
		params := zero["json"].(map[string]any)

		var payload any
		var page int
		if r.URL.Path == "/market.getModelCategories" {
			payload = []Category{{Category: "zhipu", Count: 2}, {Category: "minimax", Count: 1}}
		} else {
			page = int(params["page"].(float64))
			items := f.pages[page]
			if items == nil {
				items = []Model{}
			}
			payload = ModelsPage{
				Items: items, CurrentPage: page, PageSize: 100,
				TotalCount: 3, TotalPages: len(f.pages),
			}
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"result": map[string]any{"data": map[string]any{"json": payload}},
		}})
	}
}

func testModel(identifier, category string) Model {
	enabled := true
	return Model{
		ID: identifier, Identifier: identifier, DisplayName: "Model " + identifier,
		Category: category, ProviderID: category, Providers: []string{category},
		ProviderCount: 1, ContextWindowTokens: 128000, Enabled: &enabled,
		Abilities: map[string]bool{"functionCall": true},
	}
}

func TestFetchAllModelsPagesThroughCatalog(t *testing.T) {
	fake := &fakeMarket{pages: map[int][]Model{
		1: {testModel("glm-5.2", "zhipu"), testModel("glm-5.3", "zhipu")},
		2: {testModel("MiniMax-M3", "minimax")},
	}}
	server := httptest.NewServer(fake.handler(t))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	first, items, err := client.FetchAllModels(context.Background())
	if err != nil {
		t.Fatalf("FetchAllModels: %v", err)
	}
	if first.TotalCount != 3 || len(items) != 3 {
		t.Fatalf("expected 3 models across 2 pages, got first=%+v items=%d", first.TotalCount, len(items))
	}
	if items[2].Identifier != "MiniMax-M3" || items[2].Category != "minimax" {
		t.Fatalf("page-2 model not decoded correctly: %+v", items[2])
	}
	if items[0].Abilities["functionCall"] != true {
		t.Fatalf("abilities not decoded: %+v", items[0].Abilities)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("expected 2 requests (one per page), got %d", len(fake.requests))
	}
}

func TestFetchModelsPageMarksUndefinedFields(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.Unmarshal([]byte(r.URL.Query().Get("input")), &captured)
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"result": map[string]any{"data": map[string]any{"json": ModelsPage{Items: []Model{}, TotalPages: 1}}},
		}})
	}))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	if _, err := client.FetchModelsPage(context.Background(), ListOptions{Category: "zhipu"}); err != nil {
		t.Fatalf("FetchModelsPage: %v", err)
	}

	zero := captured["0"].(map[string]any)
	params := zero["json"].(map[string]any)
	if params["category"] != "zhipu" {
		t.Fatalf("category filter not sent: %v", params)
	}
	if _, hasMeta := zero["meta"]; !hasMeta {
		t.Fatalf("undefined fields must be listed in meta.values")
	}
	meta := zero["meta"].(map[string]any)
	values := meta["values"].(map[string]any)
	for _, field := range []string{"q", "order", "sort"} {
		if _, ok := values[field]; !ok {
			t.Fatalf("field %s must be marked undefined, got meta %v", field, values)
		}
	}
	if _, ok := values["category"]; ok {
		t.Fatalf("category is set, must not be marked undefined")
	}
}

func TestFetchCategoriesDecodesCounts(t *testing.T) {
	fake := &fakeMarket{}
	server := httptest.NewServer(fake.handler(t))
	defer server.Close()

	client := NewClient()
	client.BaseURL = server.URL
	categories, err := client.FetchCategories(context.Background())
	if err != nil {
		t.Fatalf("FetchCategories: %v", err)
	}
	if len(categories) != 2 || categories[0].Category != "zhipu" || categories[0].Count != 2 {
		t.Fatalf("unexpected categories: %+v", categories)
	}
}

func TestCategoriesFromModelsSortsAndCounts(t *testing.T) {
	items := []Model{
		testModel("a1", "zhipu"), testModel("a2", "zhipu"), testModel("b1", "minimax"),
	}
	categories := CategoriesFromModels(items)
	if len(categories) != 2 || categories[0].Category != "minimax" || categories[0].Count != 1 {
		t.Fatalf("categories not sorted/counted: %+v", categories)
	}
	if categories[1].Category != "zhipu" || categories[1].Count != 2 {
		t.Fatalf("categories not sorted/counted: %+v", categories)
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

func TestSettingsNormalization(t *testing.T) {
	if got := (Settings{}).Normalized().Storage; got != StorageSQLite {
		t.Fatalf("empty storage must default to sqlite, got %q", got)
	}
	if got := (Settings{Storage: "file"}).Normalized(); !got.UsesFile() || got.UsesSQLite() {
		t.Fatalf("file storage misconfigured: %+v", got)
	}
	both := Settings{Storage: "BOTH", SourceURL: "", Locale: ""}
	if !both.Normalized().UsesSQLite() || !both.Normalized().UsesFile() {
		t.Fatal("both storage must write to sqlite and file")
	}
	if got := both.Normalized().SourceURL; got != DefaultBaseURL {
		t.Fatalf("empty source must default to lobehub, got %q", got)
	}
	if got := (Settings{Storage: "nonsense"}).Normalized().Storage; got != StorageSQLite {
		t.Fatalf("invalid storage must fall back to sqlite, got %q", got)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AISW_MARKET_DIR", dir)

	snapshot := &Snapshot{
		Source: "test", Locale: "zh-CN", TotalCount: 1, ItemCount: 1, CategoryCount: 1,
		Items: []Model{testModel("glm-5.2", "zhipu")},
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
