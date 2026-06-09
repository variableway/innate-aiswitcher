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

		seeds := []struct {
			slug    string
			name    string
			binary  string
			adapter string
		}{
			{slug: "hermes", name: "Hermes Agent", binary: "hermes", adapter: "openai_env"},
			{slug: "openclaw", name: "OpenClaw", binary: "openclaw", adapter: "openai_env"},
		}

		for _, seed := range seeds {
			existing, err := txApp.FindFirstRecordByFilter("agents", "slug={:slug}", dbx.Params{"slug": seed.slug})
			if err == nil && existing != nil {
				continue
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			record := core.NewRecord(agents)
			record.Set("slug", seed.slug)
			record.Set("name", seed.name)
			record.Set("binary", seed.binary)
			record.Set("adapter", seed.adapter)
			record.Set("active", true)
			if err := txApp.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(txApp core.App) error {
		for _, slug := range []string{"hermes", "openclaw"} {
			record, err := txApp.FindFirstRecordByFilter("agents", "slug={:slug}", dbx.Params{"slug": slug})
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				return err
			}
			if err := txApp.Delete(record); err != nil {
				return err
			}
		}
		return nil
	})
}
