package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

// 1788931362_market_models_collection.go
//
// Adds the market_models collection: a local snapshot of the lobehub model
// market catalog fetched via POST /api/aisw/market/fetch. The full model JSON
// is kept in the raw field; the other columns exist for filtered listing.
// Catalog models are reference data — importing one into a vendor provider
// goes through the providers.models list (shared API key), never this table.
func init() {
	migrations.Register(func(txApp core.App) error {
		marketModels := core.NewBaseCollection("market_models")
		allowPublicRead(marketModels)
		marketModels.Fields.Add(
			&core.TextField{Name: "identifier", Required: true, Presentable: true},
			&core.TextField{Name: "display_name", Presentable: true},
			&core.TextField{Name: "category"},
			&core.TextField{Name: "provider_id"},
			&core.JSONField{Name: "providers"},
			&core.NumberField{Name: "context_window_tokens"},
			&core.BoolField{Name: "enabled"},
			&core.JSONField{Name: "raw"},
		)
		marketModels.AddIndex("idx_market_models_identifier", true, "identifier", "")
		marketModels.AddIndex("idx_market_models_category", false, "category", "")
		return txApp.Save(marketModels)
	}, func(txApp core.App) error {
		collection, err := txApp.FindCollectionByNameOrId("market_models")
		if err != nil {
			return err
		}
		return txApp.Delete(collection)
	})
}
