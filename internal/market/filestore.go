package market

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/variableway/innate-aiswitcher/internal/safefile"
)

// Snapshot is the on-disk catalog file format (market/models.json): the
// fetched model list plus fetch metadata.
type Snapshot struct {
	Source        string    `json:"source"`
	FetchedAt     time.Time `json:"fetchedAt"`
	Locale        string    `json:"locale"`
	TotalCount    int       `json:"totalCount"`
	ItemCount     int       `json:"itemCount"`
	CategoryCount int       `json:"categoryCount"`
	Items         []Model   `json:"items"`
}

// Dir is the market data directory (~/.innate-aiswitcher/market by default;
// AISW_MARKET_DIR overrides).
func Dir() (string, error) {
	if dir := os.Getenv("AISW_MARKET_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".innate-aiswitcher", "market"), nil
}

// FilePath is the catalog snapshot location inside Dir.
func FilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "models.json"), nil
}

// SaveFile atomically writes the snapshot (0o600) and returns its path.
func SaveFile(snapshot *Snapshot) (string, error) {
	path, err := FilePath()
	if err != nil {
		return "", err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", err
		}
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	if err := safefile.Write(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// LoadFile reads the snapshot; it returns (nil, nil) when no snapshot exists.
func LoadFile() (*Snapshot, error) {
	path, err := FilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("invalid market snapshot %s: %w", path, err)
	}
	return &snapshot, nil
}
