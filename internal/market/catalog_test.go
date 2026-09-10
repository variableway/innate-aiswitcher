package market

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeCatalogStore is an in-memory FetchStore for the catalog helpers.
type fakeCatalogStore struct {
	settings map[string]any
	models   []Model
	lastSave []Model
}

func (f *fakeCatalogStore) GetSetting(key string, dest any) (bool, error) {
	value, ok := f.settings[key]
	if !ok {
		return false, nil
	}
	data, _ := json.Marshal(value)
	return true, json.Unmarshal(data, dest)
}

func (f *fakeCatalogStore) SetSetting(key string, value any) error {
	if f.settings == nil {
		f.settings = map[string]any{}
	}
	f.settings[key] = value
	return nil
}

func (f *fakeCatalogStore) ListMarketModels(filter Filter) ([]Model, error) {
	query := strings.ToLower(filter.Query)
	result := []Model{}
	for _, model := range f.models {
		if filter.Category != "" && model.Category != filter.Category {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(model.Identifier), query) &&
			!strings.Contains(strings.ToLower(model.DisplayName), query) {
			continue
		}
		result = append(result, model)
	}
	return result, nil
}

func (f *fakeCatalogStore) ReplaceMarketModels(items []Model) error {
	f.models = append([]Model{}, items...)
	f.lastSave = items
	return nil
}

func TestLoadCatalogDefaultsToEmptyWithoutData(t *testing.T) {
	store := &fakeCatalogStore{settings: map[string]any{}}
	catalog, err := LoadCatalog(store, Filter{})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if catalog.HasData || len(catalog.Items) != 0 {
		t.Fatalf("expected empty catalog, got %+v", catalog)
	}
	if catalog.Storage != StorageSQLite {
		t.Fatalf("default storage must be sqlite, got %q", catalog.Storage)
	}
}

func TestLoadCatalogReadsSQLiteBackendWithMeta(t *testing.T) {
	store := &fakeCatalogStore{
		settings: map[string]any{
			MetaKey: Meta{FetchedAt: time.Unix(1788900000, 0), ItemCount: 2},
		},
		models: []Model{testModel("glm-5.2", "zhipu"), testModel("MiniMax-M3", "minimax")},
	}
	catalog, err := LoadCatalog(store, Filter{Query: "glm"})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if !catalog.HasData || len(catalog.Items) != 1 || catalog.Items[0].Identifier != "glm-5.2" {
		t.Fatalf("unexpected catalog: hasData=%v items=%+v", catalog.HasData, catalog.Items)
	}
	if !catalog.FetchedAt.Equal(time.Unix(1788900000, 0)) {
		t.Fatalf("fetchedAt must come from the meta KV row, got %v", catalog.FetchedAt)
	}
}

func TestLoadCatalogReadsFileBackend(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AISW_MARKET_DIR", dir)
	if _, err := SaveFile(&Snapshot{
		FetchedAt: time.Unix(1788900100, 0), Items: []Model{testModel("glm-5.2", "zhipu")},
	}); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	store := &fakeCatalogStore{settings: map[string]any{
		SettingsKey: Settings{Storage: StorageFile},
	}}

	catalog, err := LoadCatalog(store, Filter{})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(catalog.Items) != 1 || catalog.Storage != StorageFile {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	if !catalog.FetchedAt.Equal(time.Unix(1788900100, 0)) {
		t.Fatalf("fetchedAt must come from the snapshot, got %v", catalog.FetchedAt)
	}
}

func TestFetchAndStorePersistsToBothBackends(t *testing.T) {
	fake := &fakeMarket{pages: map[int][]Model{
		1: {testModel("glm-5.2", "zhipu"), testModel("glm-5.3", "zhipu")},
		2: {testModel("MiniMax-M3", "minimax")},
	}}
	server := httptest.NewServer(fake.handler(t))
	defer server.Close()

	dir := t.TempDir()
	t.Setenv("AISW_MARKET_DIR", dir)
	store := &fakeCatalogStore{settings: map[string]any{
		SettingsKey: Settings{Storage: StorageBoth, SourceURL: server.URL},
	}}

	result, err := FetchAndStore(context.Background(), store, LoadSettings(store), nil)
	if err != nil {
		t.Fatalf("FetchAndStore: %v", err)
	}
	if result.Fetched != 3 || result.Categories != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Storages) != 2 || result.Storages[0] != StorageSQLite {
		t.Fatalf("both backends must be reported, got %v", result.Storages)
	}
	if len(store.lastSave) != 3 {
		t.Fatalf("sqlite snapshot must hold 3 models, got %d", len(store.lastSave))
	}
	snapshot, err := LoadFile()
	if err != nil || snapshot == nil || snapshot.ItemCount != 3 {
		t.Fatalf("file snapshot wrong: %v %+v", err, snapshot)
	}

	var meta Meta
	if ok, _ := store.GetSetting(MetaKey, &meta); !ok || meta.ItemCount != 3 {
		t.Fatalf("meta KV must record the fetch, got %+v ok=%v", meta, ok)
	}

	// reads flow through the configured backend
	catalog, err := LoadCatalog(store, Filter{Category: "minimax"})
	if err != nil || len(catalog.Items) != 1 {
		t.Fatalf("post-fetch read failed: %v %+v", err, catalog)
	}
}
