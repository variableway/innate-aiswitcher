package templates

import "testing"

func TestProviderPresetsIncludeMinimaxURLChoices(t *testing.T) {
	preset, err := FindPreset("minimax")
	if err != nil {
		t.Fatal(err)
	}
	if len(preset.URLOptions) != 2 {
		t.Fatalf("expected minimax to have 2 URL options, got %d", len(preset.URLOptions))
	}

	protocols := map[string]string{}
	models := map[string]string{}
	endpoints := map[string]map[string]string{}
	for _, option := range preset.URLOptions {
		protocols[option.Slug] = option.APIProtocol
		models[option.Slug] = option.DefaultModel
		endpoints[option.Slug] = option.Endpoints
	}
	if protocols["openai"] != "openai_chat" || protocols["claude"] != "anthropic" {
		t.Fatalf("unexpected protocols: %+v", protocols)
	}
	if models["openai"] == "" || models["claude"] == "" {
		t.Fatalf("preset options must define default models: %+v", models)
	}
	if endpoints["openai"]["chat_completions"] == "" || endpoints["claude"]["messages"] == "" {
		t.Fatalf("preset options must define protocol endpoints: %+v", endpoints)
	}
}

func TestConfigExampleIsEmbedded(t *testing.T) {
	content, err := ConfigExample()
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("expected embedded config template")
	}
}
