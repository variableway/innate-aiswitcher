package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

// 1780993000_provider_variants_and_focus_agents.go
//
// Introduces the vendor-centric provider model ("LLM Provider Config"):
//   - providers.models    (JSON list)  models share the vendor's single API key
//   - providers.variants  (JSON map)   per-protocol endpoints (anthropic /
//     openai_responses / openai_chat), so one
//     vendor row serves every agent adapter
//
// Narrows focus to the claude / codex / opencode agents:
//   - deletes the gemini, kimi, trae, hermes, openclaw agent rows (their
//     profiles cascade-delete with them)
//   - trims providers.api_protocol and agents.adapter select values
func init() {
	migrations.Register(func(txApp core.App) error {
		providers, err := txApp.FindCollectionByNameOrId("providers")
		if err != nil {
			return err
		}
		if providers.Fields.GetByName("models") == nil {
			providers.Fields.Add(&core.JSONField{Name: "models"})
		}
		if providers.Fields.GetByName("variants") == nil {
			providers.Fields.Add(&core.JSONField{Name: "variants"})
		}
		if field, ok := providers.Fields.GetByName("api_protocol").(*core.SelectField); ok {
			field.Values = []string{"anthropic", "openai_chat", "openai_responses"}
		}
		if err := txApp.Save(providers); err != nil {
			return err
		}

		agents, err := txApp.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}
		if field, ok := agents.Fields.GetByName("adapter").(*core.SelectField); ok {
			field.Values = []string{"claude", "codex", "openai_env"}
		}
		if err := txApp.Save(agents); err != nil {
			return err
		}

		nonFocus := []string{"gemini", "kimi", "trae", "hermes", "openclaw"}
		for _, slug := range nonFocus {
			records, err := txApp.FindRecordsByFilter("agents", "slug={:slug}", "", 0, 0, dbx.Params{"slug": slug})
			if err != nil {
				return err
			}
			for _, record := range records {
				if err := txApp.Delete(record); err != nil {
					return err
				}
			}
		}
		return nil
	}, func(txApp core.App) error {
		// The models/variants fields and select-value trim are additive/
		// cosmetic; deleted agent rows are not resurrected on downgrade.
		providers, err := txApp.FindCollectionByNameOrId("providers")
		if err != nil {
			return err
		}
		if field, ok := providers.Fields.GetByName("api_protocol").(*core.SelectField); ok {
			field.Values = []string{"anthropic", "openai_chat", "openai_responses", "gemini_native", "generic"}
		}
		if err := txApp.Save(providers); err != nil {
			return err
		}
		agents, err := txApp.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}
		if field, ok := agents.Fields.GetByName("adapter").(*core.SelectField); ok {
			field.Values = []string{"claude", "codex", "gemini", "openai_env"}
		}
		return txApp.Save(agents)
	})
}
