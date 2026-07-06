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

// UpsertItem creates or updates an item.
func (s *Store) UpsertItem(input Item) (*Item, error) {
	input.Slug = normalizeSlug(input.Slug)
	if input.Slug == "" {
		return nil, fmt.Errorf("item slug is required")
	}
	if input.Name == "" {
		input.Name = input.Slug
	}

	record, err := s.findRecordBySlug("items", input.Slug)
	if err != nil {
		return nil, err
	}
	if record == nil {
		collection, err := s.app.FindCollectionByNameOrId("items")
		if err != nil {
			return nil, err
		}
		record = core.NewRecord(collection)
	}

	record.Set("slug", input.Slug)
	record.Set("name", input.Name)
	record.Set("description", input.Description)
	if !input.Active {
		input.Active = true
	}
	record.Set("active", input.Active)

	if err := s.app.Save(record); err != nil {
		return nil, err
	}
	return recordToItem(record), nil
}

// ListItems returns all items sorted by slug.
func (s *Store) ListItems() ([]Item, error) {
	records, err := s.app.FindRecordsByFilter("items", "", "slug", 0, 0)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(records))
	for _, record := range records {
		items = append(items, *recordToItem(record))
	}
	return items, nil
}

// GetItem returns a single item by slug.
func (s *Store) GetItem(slug string) (*Item, error) {
	record, err := s.findRecordBySlug("items", slug)
	if err != nil || record == nil {
		return nil, err
	}
	return recordToItem(record), nil
}

// DeleteItem removes an item by slug.
func (s *Store) DeleteItem(slug string) error {
	record, err := s.findRecordBySlug("items", slug)
	if err != nil || record == nil {
		return err
	}
	return s.app.Delete(record)
}

// --- helpers ---

func (s *Store) findRecordBySlug(collection, slug string) (*core.Record, error) {
	slug = normalizeSlug(slug)
	if slug == "" {
		return nil, nil
	}
	record, err := s.app.FindFirstRecordByFilter(
		collection, "slug={:slug}", dbx.Params{"slug": slug},
	)
	if err != nil {
		return nil, nil
	}
	return record, nil
}

func recordToItem(record *core.Record) *Item {
	return &Item{
		ID:          record.Id,
		Slug:        record.GetString("slug"),
		Name:        record.GetString("name"),
		Description: record.GetString("description"),
		Active:      record.GetBool("active"),
	}
}

func normalizeSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}
