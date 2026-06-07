package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(func(txApp core.App) error {
		providers, err := txApp.FindCollectionByNameOrId("providers")
		if err != nil {
			return err
		}
		if providers.Fields.GetByName("endpoints") != nil {
			return nil
		}
		providers.Fields.Add(&core.JSONField{Name: "endpoints"})
		return txApp.Save(providers)
	}, func(txApp core.App) error {
		providers, err := txApp.FindCollectionByNameOrId("providers")
		if err != nil {
			return err
		}
		providers.Fields.RemoveByName("endpoints")
		return txApp.Save(providers)
	})
}
