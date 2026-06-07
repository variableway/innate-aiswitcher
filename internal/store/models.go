package store

import "encoding/json"

type Provider struct {
	ID           string                 `json:"id" toml:"-"`
	Slug         string                 `json:"slug" toml:"slug"`
	Name         string                 `json:"name" toml:"name"`
	BaseURL      string                 `json:"base_url" toml:"base_url"`
	APIKey       string                 `json:"api_key,omitempty" toml:"api_key,omitempty"`
	APIProtocol  string                 `json:"api_protocol" toml:"api_protocol"`
	DefaultModel string                 `json:"default_model,omitempty" toml:"default_model,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty" toml:"headers,omitempty"`
	Endpoints    map[string]string      `json:"endpoints,omitempty" toml:"endpoints,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty" toml:"capabilities,omitempty"`
	Notes        string                 `json:"notes,omitempty" toml:"notes,omitempty"`
	Active       bool                   `json:"active" toml:"active"`
}

type Agent struct {
	ID      string            `json:"id" toml:"-"`
	Slug    string            `json:"slug" toml:"slug"`
	Name    string            `json:"name" toml:"name"`
	Binary  string            `json:"binary" toml:"binary"`
	Adapter string            `json:"adapter" toml:"adapter"`
	EnvMap  map[string]string `json:"env_map,omitempty" toml:"env_map,omitempty"`
	Active  bool              `json:"active" toml:"active"`
}

type Profile struct {
	ID              string                 `json:"id" toml:"-"`
	Slug            string                 `json:"slug" toml:"slug"`
	Name            string                 `json:"name" toml:"name"`
	AgentID         string                 `json:"agent_id" toml:"-"`
	AgentSlug       string                 `json:"agent" toml:"agent"`
	ProviderID      string                 `json:"provider_id" toml:"-"`
	ProviderSlug    string                 `json:"provider" toml:"provider"`
	Model           string                 `json:"model,omitempty" toml:"model,omitempty"`
	ConfigOverrides map[string]interface{} `json:"config_overrides,omitempty" toml:"config_overrides,omitempty"`
	EnvOverrides    map[string]string      `json:"env_overrides,omitempty" toml:"env_overrides,omitempty"`
	DefaultArgs     string                 `json:"default_args,omitempty" toml:"default_args,omitempty"`
	IsDefault       bool                   `json:"is_default" toml:"is_default"`
}

type Binding struct {
	ID           string `json:"id" toml:"-"`
	Scope        string `json:"scope" toml:"scope"`
	ProjectPath  string `json:"project_path,omitempty" toml:"project_path,omitempty"`
	AgentSlug    string `json:"agent,omitempty" toml:"agent,omitempty"`
	ProviderSlug string `json:"provider,omitempty" toml:"provider,omitempty"`
	ProfileSlug  string `json:"profile,omitempty" toml:"profile,omitempty"`
}

type LaunchHistory struct {
	AgentID    string
	ProviderID string
	ProfileID  string
	CWD        string
	Command    string
	Terminal   string
	Status     string
	Error      string
}

func decodeJSONMap[T any](value any) T {
	var out T
	if value == nil {
		return out
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return out
	}
	_ = json.Unmarshal(bytes, &out)
	return out
}
