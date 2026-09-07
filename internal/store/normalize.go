package store

import (
	"strings"

	"github.com/pocketbase/dbx"
)

// Legacy vendors were stored as per-protocol rows with names like
// "MiniMax Claude Code-compatible" and slugs like "minimax-claude". This
// table merges them back into single vendor rows named after the vendor
// itself (e.g. "MiniMax", "Xiaomi MiMo").
type legacyVendorGroup struct {
	target  string
	aliases []string
}

var legacyVendorGroups = []legacyVendorGroup{
	{target: "minimax", aliases: []string{"minimax", "minimax-openai", "minimax-claude", "minimax-codex"}},
	{target: "glm", aliases: []string{"glm", "volcengine", "volcengine-openai", "volcengine-claude", "volcengine-codex"}},
	{target: "xiaomi", aliases: []string{"xiaomi", "xiaomi-openai", "xiaomi-claude", "xiaomi-codex"}},
	{target: "kimi", aliases: []string{"kimi", "kimi-openai", "kimi-claude", "kimi-codex"}},
	{target: "deepseek", aliases: []string{"deepseek", "deepseek-openai", "deepseek-claude", "deepseek-codex"}},
	{target: "openai", aliases: []string{"openai", "openai-claude", "openai-codex"}},
	{target: "anthropic", aliases: []string{"anthropic", "anthropic-claude", "anthropic-openai"}},
}

// NormalizeLegacyVendors merges legacy per-protocol provider rows
// ("minimax-claude", "minimax-codex", …) into one vendor row per group
// ("minimax" with the bundled preset's variants and clean name). API keys,
// configured models and default models are preserved; profiles referencing
// the legacy rows are re-pointed to the merged row. presetBySlug supplies
// the vendor template (from templates.AllPresets) for variants/name/models.
// It returns the list of merged vendor slugs (empty when nothing to do).
func (s *Store) NormalizeLegacyVendors(presetBySlug map[string]Provider) ([]string, error) {
	merged := []string{}
	for _, group := range legacyVendorGroups {
		did, err := s.mergeGroup(group, presetBySlug[group.target])
		if err != nil {
			return merged, err
		}
		if did {
			merged = append(merged, group.target)
		}
	}
	return merged, nil
}

func (s *Store) mergeGroup(group legacyVendorGroup, preset Provider) (bool, error) {
	records, err := s.app.FindRecordsByFilter("providers", "", "slug", 0, 0)
	if err != nil {
		return false, err
	}

	aliasSet := map[string]bool{}
	for _, alias := range group.aliases {
		aliasSet[alias] = true
	}

	var legacy []*recordSnapshot
	for _, record := range records {
		slug := record.GetString("slug")
		if !aliasSet[slug] {
			continue
		}
		legacy = append(legacy, &recordSnapshot{
			id:           record.Id,
			slug:         slug,
			suffixed:     strings.Contains(slug, "-"),
			apiKey:       record.GetString("api_key"),
			defaultModel: record.GetString("default_model"),
			models:       decodeJSONSlice[string](record.Get("models")),
			baseURL:      record.GetString("base_url"),
			apiProtocol:  record.GetString("api_protocol"),
		})
	}
	if len(legacy) == 0 {
		return false, nil
	}
	// Nothing to merge when a single bare row already exists and no
	// suffixed siblings remain.
	suffixed := 0
	for _, row := range legacy {
		if row.suffixed {
			suffixed++
		}
	}
	if suffixed == 0 && len(legacy) == 1 {
		return false, nil
	}

	// Assemble the merged vendor row from the bundled preset, keeping the
	// user's stored key, models and default model from the legacy rows.
	mergedRow := preset
	presetDefault := preset.DefaultModel
	if mergedRow.Slug == "" {
		// No bundled preset for this vendor — fall back to the first legacy
		// row's single-protocol endpoint so the schema stays satisfied.
		mergedRow = Provider{Slug: group.target, BaseURL: legacy[0].baseURL, APIProtocol: legacy[0].apiProtocol}
		presetDefault = legacy[0].defaultModel
	}
	mergedRow.Slug = group.target
	mergedRow.DefaultModel = ""
	for _, row := range legacy {
		if row.apiKey != "" {
			mergedRow.APIKey = row.apiKey
		}
		mergedRow.Models = append(mergedRow.Models, row.models...)
		if row.defaultModel != "" && mergedRow.DefaultModel == "" {
			mergedRow.DefaultModel = row.defaultModel
		}
	}
	if mergedRow.DefaultModel == "" {
		mergedRow.DefaultModel = presetDefault
	}
	if mergedRow.DefaultModel == "" && len(mergedRow.Models) > 0 {
		mergedRow.DefaultModel = mergedRow.Models[0]
	}

	var keepID string
	saved, err := s.UpsertProvider(mergedRow)
	if err != nil {
		return false, err
	}
	keepID = saved.ID

	// Re-point profiles from legacy rows to the merged vendor row.
	for _, row := range legacy {
		if row.id == keepID {
			continue
		}
		profiles, err := s.app.FindRecordsByFilter("profiles", "provider={:provider}", "", 0, 0, dbx.Params{"provider": row.id})
		if err != nil {
			return false, err
		}
		for _, profile := range profiles {
			profile.Set("provider", keepID)
			if err := s.app.Save(profile); err != nil {
				return false, err
			}
		}
	}

	// Drop the suffixed legacy rows (and a bare row only when the preset
	// upsert produced a different record — the bare row was replaced by
	// matching slug, so only suffixed siblings remain to delete).
	for _, row := range legacy {
		if row.id == keepID {
			continue
		}
		record, err := s.app.FindRecordById("providers", row.id)
		if err != nil {
			continue
		}
		if err := s.app.Delete(record); err != nil {
			return false, err
		}
	}
	return true, nil
}

type recordSnapshot struct {
	id           string
	slug         string
	suffixed     bool
	apiKey       string
	defaultModel string
	models       []string
	baseURL      string
	apiProtocol  string
}
