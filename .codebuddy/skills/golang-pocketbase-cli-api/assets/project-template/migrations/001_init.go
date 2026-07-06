package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(
		// UP
		func(txApp core.App) error {
			items := core.NewBaseCollection("items")
			setPublicRead(items)
			items.Fields.Add(
				&core.TextField{Name: "slug", Required: true, Presentable: true},
				&core.TextField{Name: "name", Required: true},
				&core.TextField{Name: "description"},
				&core.BoolField{Name: "active"},
			)
			items.AddIndex("idx_items_slug", true, "slug", "")
			return txApp.Save(items)
		},
		// DOWN
		func(txApp core.App) error {
			coll, err := txApp.FindCollectionByNameOrId("items")
			if err == nil {
				return txApp.Delete(coll)
			}
			return nil
		},
	)
}

func setPublicRead(collection *core.Collection) {
	rule := ""
	collection.ListRule = &rule
	collection.ViewRule = &rule
}
