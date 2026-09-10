package market

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Settings KV keys backing the market feature ("按照配置执行": storage backend +
// source) and its fetch state. Shared by the REST routes and the TUI.
const (
	SettingsKey = "market"
	MetaKey     = "market.meta"
)

// Filter narrows catalog reads: Category matches the vendor slug exactly,
// Query is a case-insensitive substring over identifier and display name.
// Both are optional. (Moved here from store so both layers share one type.)
type Filter struct {
	Category string
	Query    string
}

// Meta records the state of the last successful fetch.
type Meta struct {
	FetchedAt     time.Time `json:"fetchedAt"`
	TotalCount    int       `json:"totalCount"`
	ItemCount     int       `json:"itemCount"`
	CategoryCount int       `json:"categoryCount"`
	Storage       string    `json:"storage"`
}

// Catalog is the read model shared by the REST routes and the TUI.
type Catalog struct {
	Items     []Model
	Storage   string
	FetchedAt time.Time
	HasData   bool
}

// FetchResult summarizes a completed fetch+persist run.
type FetchResult struct {
	Fetched    int
	TotalCount int
	Categories int
	Storages   []string
	FetchedAt  time.Time
}

// SettingsStore is the minimal settings access the catalog helpers need;
// *store.Store satisfies it.
type SettingsStore interface {
	GetSetting(key string, dest any) (bool, error)
	SetSetting(key string, value any) error
}

// CatalogStore adds catalog reads over SettingsStore.
type CatalogStore interface {
	SettingsStore
	ListMarketModels(filter Filter) ([]Model, error)
}

// FetchStore adds snapshot persistence over CatalogStore.
type FetchStore interface {
	CatalogStore
	ReplaceMarketModels(items []Model) error
}

// LoadSettings returns the market settings with defaults filled in.
func LoadSettings(s SettingsStore) Settings {
	settings := DefaultSettings()
	_, _ = s.GetSetting(SettingsKey, &settings)
	return settings.Normalized()
}

// LoadCatalog reads the catalog from the backend configured in the market
// settings (sqlite preferred, file snapshot otherwise).
func LoadCatalog(s CatalogStore, filter Filter) (*Catalog, error) {
	settings := LoadSettings(s)
	catalog := &Catalog{Storage: settings.Storage}

	var meta Meta
	if ok, _ := s.GetSetting(MetaKey, &meta); ok {
		catalog.FetchedAt = meta.FetchedAt
		catalog.HasData = true
	}

	if settings.UsesSQLite() {
		items, err := s.ListMarketModels(filter)
		if err != nil {
			return nil, err
		}
		catalog.Items = items
		return catalog, nil
	}

	snapshot, err := LoadFile()
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		catalog.Items = []Model{}
		return catalog, nil
	}
	catalog.FetchedAt = snapshot.FetchedAt
	catalog.HasData = true
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, item := range snapshot.Items {
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(item.Identifier), query) &&
			!strings.Contains(strings.ToLower(item.DisplayName), query) {
			continue
		}
		catalog.Items = append(catalog.Items, item)
	}
	return catalog, nil
}

// FetchAndStore pulls the whole catalog from the configured source and
// persists it to the configured backend(s), updating the fetch meta. An nil
// httpClient falls back to the default client.
func FetchAndStore(ctx context.Context, s FetchStore, settings Settings, httpClient *http.Client) (*FetchResult, error) {
	client := NewClient()
	if httpClient != nil {
		client.HTTP = httpClient
	}
	client.BaseURL = settings.SourceURL
	client.Locale = settings.Locale

	first, items, err := client.FetchAllModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("market fetch failed: %w", err)
	}
	categories := CategoriesFromModels(items)

	storages := make([]string, 0, 2)
	if settings.UsesSQLite() {
		if err := s.ReplaceMarketModels(items); err != nil {
			return nil, fmt.Errorf("sqlite persist failed: %w", err)
		}
		storages = append(storages, StorageSQLite)
	}
	if settings.UsesFile() {
		path, err := SaveFile(&Snapshot{
			Source:        settings.SourceURL,
			FetchedAt:     time.Now(),
			Locale:        settings.Locale,
			TotalCount:    first.TotalCount,
			ItemCount:     len(items),
			CategoryCount: len(categories),
			Items:         items,
		})
		if err != nil {
			return nil, fmt.Errorf("file persist failed: %w", err)
		}
		storages = append(storages, StorageFile+" ("+path+")")
	}

	meta := Meta{
		FetchedAt:     time.Now(),
		TotalCount:    first.TotalCount,
		ItemCount:     len(items),
		CategoryCount: len(categories),
		Storage:       settings.Storage,
	}
	if err := s.SetSetting(MetaKey, meta); err != nil {
		return nil, err
	}
	return &FetchResult{
		Fetched:    len(items),
		TotalCount: first.TotalCount,
		Categories: len(categories),
		Storages:   storages,
		FetchedAt:  meta.FetchedAt,
	}, nil
}
