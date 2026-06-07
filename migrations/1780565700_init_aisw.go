package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(func(txApp core.App) error {
		providers := core.NewBaseCollection("providers")
		allowPublicRead(providers)
		providers.Fields.Add(
			&core.TextField{Name: "slug", Required: true, Presentable: true},
			&core.TextField{Name: "name", Required: true},
			&core.URLField{Name: "base_url", Required: true},
			&core.TextField{Name: "api_key", Hidden: true},
			&core.SelectField{Name: "api_protocol", Required: true, Values: []string{"anthropic", "openai_chat", "openai_responses", "gemini_native", "generic"}},
			&core.TextField{Name: "default_model"},
			&core.JSONField{Name: "headers"},
			&core.JSONField{Name: "endpoints"},
			&core.JSONField{Name: "capabilities"},
			&core.TextField{Name: "notes"},
			&core.BoolField{Name: "active"},
		)
		providers.AddIndex("idx_providers_slug", true, "slug", "")
		if err := txApp.Save(providers); err != nil {
			return err
		}

		agents := core.NewBaseCollection("agents")
		allowPublicRead(agents)
		agents.Fields.Add(
			&core.TextField{Name: "slug", Required: true, Presentable: true},
			&core.TextField{Name: "name", Required: true},
			&core.TextField{Name: "binary", Required: true},
			&core.SelectField{Name: "adapter", Required: true, Values: []string{"claude", "codex", "gemini", "openai_env"}},
			&core.JSONField{Name: "env_map"},
			&core.BoolField{Name: "active"},
		)
		agents.AddIndex("idx_agents_slug", true, "slug", "")
		if err := txApp.Save(agents); err != nil {
			return err
		}

		profiles := core.NewBaseCollection("profiles")
		allowPublicRead(profiles)
		profiles.Fields.Add(
			&core.TextField{Name: "slug", Required: true, Presentable: true},
			&core.TextField{Name: "name", Required: true},
			&core.RelationField{Name: "agent", Required: true, CollectionId: agents.Id, CascadeDelete: true, MaxSelect: 1},
			&core.RelationField{Name: "provider", Required: true, CollectionId: providers.Id, CascadeDelete: true, MaxSelect: 1},
			&core.TextField{Name: "model"},
			&core.JSONField{Name: "config_overrides"},
			&core.JSONField{Name: "env_overrides"},
			&core.TextField{Name: "default_args"},
			&core.BoolField{Name: "is_default"},
		)
		profiles.AddIndex("idx_profiles_slug", true, "slug", "")
		profiles.AddIndex("idx_profiles_agent_provider", false, "agent, provider", "")
		if err := txApp.Save(profiles); err != nil {
			return err
		}

		bindings := core.NewBaseCollection("bindings")
		bindings.Fields.Add(
			&core.SelectField{Name: "scope", Required: true, Values: []string{"global", "project", "session"}},
			&core.TextField{Name: "project_path"},
			&core.RelationField{Name: "agent", CollectionId: agents.Id, MaxSelect: 1},
			&core.RelationField{Name: "provider", CollectionId: providers.Id, MaxSelect: 1},
			&core.RelationField{Name: "profile", CollectionId: profiles.Id, MaxSelect: 1},
		)
		bindings.AddIndex("idx_bindings_scope_project_agent", false, "scope, project_path, agent", "")
		if err := txApp.Save(bindings); err != nil {
			return err
		}

		launchHistory := core.NewBaseCollection("launch_history")
		launchHistory.Fields.Add(
			&core.RelationField{Name: "agent", CollectionId: agents.Id, MaxSelect: 1},
			&core.RelationField{Name: "provider", CollectionId: providers.Id, MaxSelect: 1},
			&core.RelationField{Name: "profile", CollectionId: profiles.Id, MaxSelect: 1},
			&core.TextField{Name: "cwd"},
			&core.TextField{Name: "command"},
			&core.TextField{Name: "terminal"},
			&core.SelectField{Name: "status", Values: []string{"planned", "started", "success", "failed"}},
			&core.TextField{Name: "error"},
		)
		if err := txApp.Save(launchHistory); err != nil {
			return err
		}

		settings := core.NewBaseCollection("settings")
		settings.Fields.Add(
			&core.TextField{Name: "key", Required: true, Presentable: true},
			&core.JSONField{Name: "value"},
		)
		settings.AddIndex("idx_settings_key", true, "key", "")
		if err := txApp.Save(settings); err != nil {
			return err
		}

		return seedAgents(txApp, agents)
	}, func(txApp core.App) error {
		for _, name := range []string{"settings", "launch_history", "bindings", "profiles", "agents", "providers"} {
			collection, err := txApp.FindCollectionByNameOrId(name)
			if err == nil {
				if err := txApp.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func allowPublicRead(collection *core.Collection) {
	rule := ""
	collection.ListRule = &rule
	collection.ViewRule = &rule
}

func seedAgents(app core.App, agents *core.Collection) error {
	seeds := []struct {
		slug    string
		name    string
		binary  string
		adapter string
	}{
		{slug: "claude", name: "Claude Code", binary: "claude", adapter: "claude"},
		{slug: "codex", name: "Codex CLI", binary: "codex", adapter: "codex"},
		{slug: "gemini", name: "Gemini CLI", binary: "gemini", adapter: "gemini"},
		{slug: "kimi", name: "Kimi CLI", binary: "kimi", adapter: "openai_env"},
		{slug: "trae", name: "Trae CLI", binary: "trae", adapter: "openai_env"},
		{slug: "opencode", name: "OpenCode", binary: "opencode", adapter: "openai_env"},
	}

	for _, seed := range seeds {
		record := core.NewRecord(agents)
		record.Set("slug", seed.slug)
		record.Set("name", seed.name)
		record.Set("binary", seed.binary)
		record.Set("adapter", seed.adapter)
		record.Set("active", true)
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}
