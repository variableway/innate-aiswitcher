package projectconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := ProjectConfig{
		Profile:  "claude-work",
		Agent:    "claude",
		Provider: "minimax-work",
	}
	if err := Write(dir, cfg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	loaded, err := Load(filepath.Join(dir, ".aiswrc"))
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Profile != "claude-work" {
		t.Errorf("profile = %q, want claude-work", loaded.Profile)
	}
	if loaded.Agent != "claude" {
		t.Errorf("agent = %q, want claude", loaded.Agent)
	}
	if loaded.Provider != "minimax-work" {
		t.Errorf("provider = %q, want minimax-work", loaded.Provider)
	}
}

func TestFindWalkUp(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	cfg := ProjectConfig{Profile: "found"}
	if err := Write(root, cfg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	found, dir, err := Find(child)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find .aiswrc, got nil")
	}
	if found.Profile != "found" {
		t.Errorf("profile = %q, want found", found.Profile)
	}
	if dir != root {
		t.Errorf("found dir = %q, want %q", dir, root)
	}
}

func TestFindNotFound(t *testing.T) {
	dir := t.TempDir()
	found, _, err := Find(dir)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil, got config")
	}
}
