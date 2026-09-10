package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/variableway/innate-aiswitcher/internal/market"
)

// ReplaceMarketModels swaps the whole market_models snapshot in one
// transaction (delete-all + insert). The full Model JSON is kept in the raw
// field so listing restores every attribute without schema churn.
func (s *Store) ReplaceMarketModels(items []market.Model) error {
	return s.RunInTransaction(func(tx *Store) error {
		existing, err := tx.app.FindRecordsByFilter("market_models", "", "", 0, 0)
		if err != nil {
			return err
		}
		for _, record := range existing {
			if err := tx.app.Delete(record); err != nil {
				return err
			}
		}
		collection, err := tx.app.FindCollectionByNameOrId("market_models")
		if err != nil {
			return err
		}
		for _, item := range items {
			raw, err := json.Marshal(item)
			if err != nil {
				return err
			}
			record := core.NewRecord(collection)
			record.Set("identifier", item.Identifier)
			record.Set("display_name", item.DisplayName)
			record.Set("category", item.Category)
			record.Set("provider_id", item.ProviderID)
			record.Set("providers", nonNilSlice(item.Providers))
			record.Set("context_window_tokens", item.ContextWindowTokens)
			record.Set("enabled", item.IsEnabled())
			record.Set("raw", json.RawMessage(raw))
			if err := tx.app.Save(record); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListMarketModels returns catalog models matching filter, ordered by
// category then identifier.
func (s *Store) ListMarketModels(filter market.Filter) ([]market.Model, error) {
	conditions := []string{}
	params := dbx.Params{}
	if filter.Category != "" {
		conditions = append(conditions, "category={:category}")
		params["category"] = filter.Category
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		// PocketBase filter syntax: ~ is a case-insensitive "contains".
		conditions = append(conditions, "(identifier ~ {:query} || display_name ~ {:query})")
		params["query"] = query
	}
	filterExpr := strings.Join(conditions, " && ")
	records, err := s.app.FindRecordsByFilter("market_models", filterExpr, "category,identifier", 0, 0, params)
	if err != nil {
		return nil, err
	}
	models := make([]market.Model, 0, len(records))
	for _, record := range records {
		var model market.Model
		raw, err := json.Marshal(record.Get("raw"))
		if err != nil {
			continue
		}
		if err := json.Unmarshal(raw, &model); err != nil {
			continue
		}
		models = append(models, model)
	}
	return models, nil
}

// MarketModelCount returns the number of stored catalog models.
func (s *Store) MarketModelCount() (int, error) {
	records, err := s.app.FindRecordsByFilter("market_models", "", "", 0, 0)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// GetSetting decodes the JSON value of a settings KV row into dest. It
// returns false when the key does not exist yet.
func (s *Store) GetSetting(key string, dest any) (bool, error) {
	record, err := s.app.FindFirstRecordByFilter("settings", "key={:key}", dbx.Params{"key": key})
	if err != nil {
		return false, nil
	}
	if record == nil {
		return false, nil
	}
	value, err := json.Marshal(record.Get("value"))
	if err != nil {
		return true, err
	}
	if err := json.Unmarshal(value, dest); err != nil {
		return true, err
	}
	return true, nil
}

// SetSetting stores value as the JSON value of a settings KV row.
func (s *Store) SetSetting(key string, value any) error {
	if key == "" {
		return fmt.Errorf("setting key is required")
	}
	record, err := s.app.FindFirstRecordByFilter("settings", "key={:key}", dbx.Params{"key": key})
	if err != nil {
		record = nil
	}
	if record == nil {
		collection, err := s.app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		record = core.NewRecord(collection)
		record.Set("key", key)
	}
	record.Set("value", value)
	return s.app.Save(record)
}
