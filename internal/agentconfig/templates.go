package agentconfig

// Placeholder templates for each agent's on-disk configuration. Values in
// <ANGLE_BRACKETS> are meant to be replaced with the vendor's endpoint, API
// key and model — the Configs page builder does this automatically via
// adapter.Preview; these templates are the manual/starting point.

// AgentTemplate is one settings template file for an agent.
type AgentTemplate struct {
	Name    string `json:"name"`  // whitelist key for write-to-disk, empty if env-only
	Label   string `json:"label"`
	Lang    string `json:"lang"`
	Content string `json:"content"`
}

// AgentTemplates returns the settings templates per agent slug.
var AgentTemplates = map[string][]AgentTemplate{
	"claude": {{
		Name:  "settings",
		Label: "settings.json",
		Lang:  "json",
		Content: `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "<YOUR_API_KEY>",
    "ANTHROPIC_API_KEY": "<YOUR_API_KEY>",
    "ANTHROPIC_BASE_URL": "<PROVIDER_BASE_URL>",
    "ANTHROPIC_MODEL": "<MODEL>",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "<MODEL>",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "<MODEL>",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "<MODEL>"
  }
}
`,
	}},
	"codex": {
		{
			Name:  "config",
			Label: "config.toml",
			Lang:  "toml",
			Content: `model = "<MODEL>"
model_provider = "<PROVIDER_NAME>"

[model_providers.<PROVIDER_NAME>]
name = "<PROVIDER_NAME>"
base_url = "<PROVIDER_BASE_URL>"
wire_api = "chat"
`,
		},
		{
			Name:  "auth",
			Label: "auth.json",
			Lang:  "json",
			Content: `{
  "OPENAI_API_KEY": "<YOUR_API_KEY>"
}
`,
		},
	},
	"opencode": {{
		Name:  "opencode",
		Label: "opencode.json",
		Lang:  "json",
		Content: `{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "<PROVIDER_KEY>": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "<PROVIDER_NAME>",
      "options": {
        "baseURL": "<PROVIDER_BASE_URL>",
        "apiKey": "<YOUR_API_KEY>"
      },
      "models": {
        "<MODEL>": {}
      }
    }
  }
}
`,
	}},
}
