package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

// 1788931500_market_models_unique_per_vendor.go
//
// The models.dev catalog lists the same model id under several vendors
// (canonical vendor plus gateways/resellers), so the market_models unique
// constraint moves from identifier alone to (identifier, provider_id).
func init() {
	migrations.Register(func(txApp core.App) error {
		collection, err := txApp.FindCollectionByNameOrId("market_models")
		if err != nil {
			return err
		}
		collection.RemoveIndex("idx_market_models_identifier")
		collection.AddIndex("idx_market_models_identifier_provider", true, "identifier, provider_id", "")
		return txApp.Save(collection)
	}, func(txApp core.App) error {
		collection, err := txApp.FindCollectionByNameOrId("market_models")
		if err != nil {
			return err
		}
		collection.RemoveIndex("idx_market_models_identifier_provider")
		collection.AddIndex("idx_market_models_identifier", true, "identifier", "")
		return txApp.Save(collection)
	})
}
