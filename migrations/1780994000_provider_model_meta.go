package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

// 1780994000_provider_model_meta.go
//
// Adds providers.model_meta (JSON map model -> {input_price, output_price,
// multimodal, note}). Vendor /models endpoints do not expose pricing or
// modality, so these annotations are maintained locally per provider row.
func init() {
	migrations.Register(func(txApp core.App) error {
		providers, err := txApp.FindCollectionByNameOrId("providers")
		if err != nil {
			return err
		}
		if providers.Fields.GetByName("model_meta") == nil {
			providers.Fields.Add(&core.JSONField{Name: "model_meta"})
		}
		return txApp.Save(providers)
	}, func(txApp core.App) error {
		// Additive field; downgrade just drops it if present.
		providers, err := txApp.FindCollectionByNameOrId("providers")
		if err != nil {
			return err
		}
		if field := providers.Fields.GetByName("model_meta"); field != nil {
			providers.Fields.RemoveByName("model_meta")
		}
		return txApp.Save(providers)
	})
}
