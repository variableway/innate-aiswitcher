package store

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type Store struct {
	app core.App
}

func New(app core.App) *Store {
	return &Store{app: app}
}

func (s *Store) RunInTransaction(fn func(*Store) error) error {
	return s.app.RunInTransaction(func(txApp core.App) error {
		return fn(New(txApp))
	})
}

func (s *Store) UpsertProvider(input Provider) (*Provider, error) {
	input.Slug = normalizeSlug(input.Slug)
	if input.Slug == "" {
		return nil, fmt.Errorf("provider slug is required")
	}
	if input.Name == "" {
		input.Name = input.Slug
	}
	if input.APIProtocol == "" {
		input.APIProtocol = "openai_chat"
	}
	if !input.Active {
		input.Active = true
	}

	record, err := s.findRecordBySlug("providers", input.Slug)
	if err != nil {
		return nil, err
	}
	if record == nil {
		collection, err := s.app.FindCollectionByNameOrId("providers")
		if err != nil {
			return nil, err
		}
		record = core.NewRecord(collection)
	}

	record.Set("slug", input.Slug)
	record.Set("name", input.Name)
	record.Set("base_url", strings.TrimRight(input.BaseURL, "/"))
	if input.APIKey != "" {
		record.Set("api_key", input.APIKey)
	}
	record.Set("api_protocol", input.APIProtocol)
	record.Set("default_model", input.DefaultModel)
	record.Set("headers", nonNilMap(input.Headers))
	record.Set("endpoints", nonNilMap(input.Endpoints))
	record.Set("capabilities", nonNilInterfaceMap(input.Capabilities))
	record.Set("notes", input.Notes)
	record.Set("active", input.Active)

	if err := s.app.Save(record); err != nil {
		return nil, err
	}
	return recordToProvider(record), nil
}

func (s *Store) ListProviders() ([]Provider, error) {
	records, err := s.app.FindRecordsByFilter("providers", "", "slug", 0, 0)
	if err != nil {
		return nil, err
	}
	providers := make([]Provider, 0, len(records))
	for _, record := range records {
		providers = append(providers, *recordToProvider(record))
	}
	return providers, nil
}

func (s *Store) GetProvider(slug string) (*Provider, error) {
	record, err := s.findRecordBySlug("providers", slug)
	if err != nil || record == nil {
		return nil, err
	}
	return recordToProvider(record), nil
}

func (s *Store) DeleteProvider(slug string) error {
	record, err := s.findRecordBySlug("providers", slug)
	if err != nil || record == nil {
		return err
	}
	return s.app.Delete(record)
}

func (s *Store) ListAgents() ([]Agent, error) {
	records, err := s.app.FindRecordsByFilter("agents", "", "slug", 0, 0)
	if err != nil {
		return nil, err
	}
	agents := make([]Agent, 0, len(records))
	for _, record := range records {
		agents = append(agents, *recordToAgent(record))
	}
	return agents, nil
}

func (s *Store) GetAgent(slug string) (*Agent, error) {
	record, err := s.findRecordBySlug("agents", slug)
	if err != nil || record == nil {
		return nil, err
	}
	return recordToAgent(record), nil
}

func (s *Store) UpsertProfile(input Profile) (*Profile, error) {
	input.Slug = normalizeSlug(input.Slug)
	if input.Slug == "" {
		return nil, fmt.Errorf("profile slug is required")
	}
	if input.AgentSlug == "" || input.ProviderSlug == "" {
		return nil, fmt.Errorf("profile requires agent and provider")
	}
	agentRecord, err := s.findRecordBySlug("agents", input.AgentSlug)
	if err != nil {
		return nil, err
	}
	if agentRecord == nil {
		return nil, fmt.Errorf("agent not found: %s", input.AgentSlug)
	}
	providerRecord, err := s.findRecordBySlug("providers", input.ProviderSlug)
	if err != nil {
		return nil, err
	}
	if providerRecord == nil {
		return nil, fmt.Errorf("provider not found: %s", input.ProviderSlug)
	}
	if input.Name == "" {
		input.Name = input.Slug
	}

	record, err := s.findRecordBySlug("profiles", input.Slug)
	if err != nil {
		return nil, err
	}
	if record == nil {
		collection, err := s.app.FindCollectionByNameOrId("profiles")
		if err != nil {
			return nil, err
		}
		record = core.NewRecord(collection)
	}

	record.Set("slug", input.Slug)
	record.Set("name", input.Name)
	record.Set("agent", agentRecord.Id)
	record.Set("provider", providerRecord.Id)
	record.Set("model", input.Model)
	record.Set("config_overrides", nonNilInterfaceMap(input.ConfigOverrides))
	record.Set("env_overrides", nonNilMap(input.EnvOverrides))
	record.Set("default_args", input.DefaultArgs)
	record.Set("is_default", input.IsDefault)

	if err := s.app.Save(record); err != nil {
		return nil, err
	}

	result := recordToProfile(record)
	result.AgentSlug = agentRecord.GetString("slug")
	result.ProviderSlug = providerRecord.GetString("slug")
	return result, nil
}

func (s *Store) ListProfiles() ([]Profile, error) {
	records, err := s.app.FindRecordsByFilter("profiles", "", "slug", 0, 0)
	if err != nil {
		return nil, err
	}
	agents, err := s.idToAgentSlug()
	if err != nil {
		return nil, err
	}
	providers, err := s.idToProviderSlug()
	if err != nil {
		return nil, err
	}
	profiles := make([]Profile, 0, len(records))
	for _, record := range records {
		profile := recordToProfile(record)
		profile.AgentSlug = agents[profile.AgentID]
		profile.ProviderSlug = providers[profile.ProviderID]
		profiles = append(profiles, *profile)
	}
	return profiles, nil
}

func (s *Store) GetProfile(slug string) (*Profile, error) {
	record, err := s.findRecordBySlug("profiles", slug)
	if err != nil || record == nil {
		return nil, err
	}
	profile := recordToProfile(record)
	if agent, err := s.recordByID("agents", profile.AgentID); err == nil && agent != nil {
		profile.AgentSlug = agent.GetString("slug")
	}
	if provider, err := s.recordByID("providers", profile.ProviderID); err == nil && provider != nil {
		profile.ProviderSlug = provider.GetString("slug")
	}
	return profile, nil
}

func (s *Store) ResolveSelector(agentSlug, selector string) (*Agent, *Provider, *Profile, error) {
	agent, err := s.GetAgent(agentSlug)
	if err != nil || agent == nil {
		return nil, nil, nil, fmt.Errorf("agent not found: %s", agentSlug)
	}

	if selector != "" {
		if profile, err := s.GetProfile(selector); err == nil && profile != nil {
			if profile.AgentSlug != agentSlug {
				return nil, nil, nil, fmt.Errorf("profile %s belongs to agent %s, not %s", selector, profile.AgentSlug, agentSlug)
			}
			provider, err := s.GetProvider(profile.ProviderSlug)
			if err != nil || provider == nil {
				return nil, nil, nil, fmt.Errorf("provider not found for profile %s", selector)
			}
			return agent, provider, profile, nil
		}

		provider, err := s.GetProvider(selector)
		if err != nil || provider == nil {
			return nil, nil, nil, fmt.Errorf("provider or profile not found: %s", selector)
		}
		return agent, provider, nil, nil
	}

	return nil, nil, nil, fmt.Errorf("selector is required for now; add project/global bindings later")
}

func (s *Store) SaveLaunchHistory(input LaunchHistory) error {
	collection, err := s.app.FindCollectionByNameOrId("launch_history")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set("agent", input.AgentID)
	record.Set("provider", input.ProviderID)
	record.Set("profile", input.ProfileID)
	record.Set("cwd", input.CWD)
	record.Set("command", input.Command)
	record.Set("terminal", input.Terminal)
	record.Set("status", input.Status)
	record.Set("error", input.Error)
	return s.app.Save(record)
}

func (s *Store) findRecordBySlug(collection string, slug string) (*core.Record, error) {
	slug = normalizeSlug(slug)
	if slug == "" {
		return nil, nil
	}
	record, err := s.app.FindFirstRecordByFilter(collection, "slug={:slug}", dbx.Params{"slug": slug})
	if err != nil {
		return nil, nil
	}
	return record, nil
}

func (s *Store) recordByID(collection, id string) (*core.Record, error) {
	if id == "" {
		return nil, nil
	}
	return s.app.FindRecordById(collection, id)
}

func (s *Store) idToAgentSlug() (map[string]string, error) {
	agents, err := s.ListAgents()
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, agent := range agents {
		result[agent.ID] = agent.Slug
	}
	return result, nil
}

func (s *Store) idToProviderSlug() (map[string]string, error) {
	providers, err := s.ListProviders()
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, provider := range providers {
		result[provider.ID] = provider.Slug
	}
	return result, nil
}

func recordToProvider(record *core.Record) *Provider {
	return &Provider{
		ID:           record.Id,
		Slug:         record.GetString("slug"),
		Name:         record.GetString("name"),
		BaseURL:      record.GetString("base_url"),
		APIKey:       record.GetString("api_key"),
		APIProtocol:  record.GetString("api_protocol"),
		DefaultModel: record.GetString("default_model"),
		Headers:      decodeJSONMap[map[string]string](record.Get("headers")),
		Endpoints:    decodeJSONMap[map[string]string](record.Get("endpoints")),
		Capabilities: decodeJSONMap[map[string]interface{}](record.Get("capabilities")),
		Notes:        record.GetString("notes"),
		Active:       record.GetBool("active"),
	}
}

func recordToAgent(record *core.Record) *Agent {
	return &Agent{
		ID:      record.Id,
		Slug:    record.GetString("slug"),
		Name:    record.GetString("name"),
		Binary:  record.GetString("binary"),
		Adapter: record.GetString("adapter"),
		EnvMap:  decodeJSONMap[map[string]string](record.Get("env_map")),
		Active:  record.GetBool("active"),
	}
}

func recordToProfile(record *core.Record) *Profile {
	return &Profile{
		ID:              record.Id,
		Slug:            record.GetString("slug"),
		Name:            record.GetString("name"),
		AgentID:         record.GetString("agent"),
		ProviderID:      record.GetString("provider"),
		Model:           record.GetString("model"),
		ConfigOverrides: decodeJSONMap[map[string]interface{}](record.Get("config_overrides")),
		EnvOverrides:    decodeJSONMap[map[string]string](record.Get("env_overrides")),
		DefaultArgs:     record.GetString("default_args"),
		IsDefault:       record.GetBool("is_default"),
	}
}

func normalizeSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func nonNilMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func nonNilInterfaceMap(value map[string]interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	return value
}
