package store

type Item struct {
	ID          string `json:"id" toml:"-"`
	Slug        string `json:"slug" toml:"slug"`
	Name        string `json:"name" toml:"name"`
	Description string `json:"description,omitempty" toml:"description,omitempty"`
	Active      bool   `json:"active" toml:"active"`
}
