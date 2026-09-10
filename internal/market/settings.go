package market

import "strings"

// Storage backends for fetched catalog snapshots.
const (
	StorageSQLite = "sqlite"
	StorageFile   = "file"
	StorageBoth   = "both"
)

// Settings controls how the market catalog feature behaves ("按照配置执行"):
// where snapshots are persisted and which catalog endpoint to pull from.
type Settings struct {
	Storage   string `json:"storage"`
	SourceURL string `json:"source_url,omitempty"`
	Locale    string `json:"locale,omitempty"`
}

// DefaultSettings returns the stock configuration: persist to the PocketBase
// SQLite database, pull from the public lobehub catalog.
func DefaultSettings() Settings {
	return Settings{Storage: StorageSQLite, SourceURL: DefaultBaseURL, Locale: DefaultLocale}
}

// Normalized returns the settings with defaults filled in and the storage
// value validated (an invalid value falls back to sqlite).
func (s Settings) Normalized() Settings {
	if s.SourceURL == "" {
		s.SourceURL = DefaultBaseURL
	}
	if s.Locale == "" {
		s.Locale = DefaultLocale
	}
	switch strings.ToLower(strings.TrimSpace(s.Storage)) {
	case StorageSQLite:
		s.Storage = StorageSQLite
	case StorageFile:
		s.Storage = StorageFile
	case StorageBoth:
		s.Storage = StorageBoth
	default:
		s.Storage = StorageSQLite
	}
	return s
}

// UsesSQLite / UsesFile report which backends a fetch must write to.
func (s Settings) UsesSQLite() bool { return s.Storage == StorageSQLite || s.Storage == StorageBoth }
func (s Settings) UsesFile() bool   { return s.Storage == StorageFile || s.Storage == StorageBoth }
