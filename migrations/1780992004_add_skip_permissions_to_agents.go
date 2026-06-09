package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

// 1780992004_add_skip_permissions_to_agents.go
//
// Adds per-agent skip-permission flag support:
//   - agents.skip_permissions_arg     (text)   CLI flag to pass, e.g. "--dangerously-skip-permissions"
//   - agents.skip_permissions_default (bool)   whether the flag is on by default
//   - profiles.skip_permissions       (text)   per-profile override: "" | "true" | "false"
//
// Seeds:
//   - claude  → --dangerously-skip-permissions, default on
//   - kimi    → --yolo, default on
//
// Other agents are left with empty arg / default off.
func init() {
	migrations.Register(func(txApp core.App) error {
		agents, err := txApp.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}
		agents.Fields.Add(
			&core.TextField{Name: "skip_permissions_arg"},
			&core.BoolField{Name: "skip_permissions_default"},
		)
		if err := txApp.Save(agents); err != nil {
			return err
		}

		profiles, err := txApp.FindCollectionByNameOrId("profiles")
		if err != nil {
			return err
		}
		profiles.Fields.Add(
			&core.TextField{Name: "skip_permissions"},
		)
		if err := txApp.Save(profiles); err != nil {
			return err
		}

		// Seed known agents. FindFirstRecordByFilter upserts via Set+Save.
		seeds := []struct {
			slug string
			arg  string
			def  bool
		}{
			{slug: "claude", arg: "--dangerously-skip-permissions", def: true},
			{slug: "kimi", arg: "--yolo", def: true},
		}
		for _, s := range seeds {
			record, err := txApp.FindFirstRecordByFilter("agents", "slug={:slug}", dbx.Params{"slug": s.slug})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if record == nil {
				record = core.NewRecord(agents)
				record.Set("slug", s.slug)
			}
			record.Set("skip_permissions_arg", s.arg)
			record.Set("skip_permissions_default", s.def)
			if err := txApp.Save(record); err != nil {
				return err
			}
		}
		return nil
	}, func(txApp core.App) error {
		// No destructive rollback — fields are additive. Just unset the seeded
		// values so a downgrade matches the pre-migration state.
		for _, slug := range []string{"claude", "kimi"} {
			record, err := txApp.FindFirstRecordByFilter("agents", "slug={:slug}", dbx.Params{"slug": slug})
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				return err
			}
			record.Set("skip_permissions_arg", "")
			record.Set("skip_permissions_default", false)
			if err := txApp.Save(record); err != nil {
				return err
			}
		}
		return nil
	})
}
