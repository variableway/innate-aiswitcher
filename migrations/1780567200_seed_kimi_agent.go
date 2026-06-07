package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(func(txApp core.App) error {
		agents, err := txApp.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}
		existing, err := txApp.FindFirstRecordByFilter("agents", "slug={:slug}", dbx.Params{"slug": "kimi"})
		if err == nil && existing != nil {
			return nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		record := core.NewRecord(agents)
		record.Set("slug", "kimi")
		record.Set("name", "Kimi CLI")
		record.Set("binary", "kimi")
		record.Set("adapter", "openai_env")
		record.Set("active", true)
		return txApp.Save(record)
	}, func(txApp core.App) error {
		record, err := txApp.FindFirstRecordByFilter("agents", "slug={:slug}", dbx.Params{"slug": "kimi"})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return txApp.Delete(record)
	})
}
