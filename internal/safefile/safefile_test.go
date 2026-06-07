package safefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesFileWithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.txt")
	data := []byte("hello safefile")

	if err := Write(path, data, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content mismatch: got %q, want %q", got, data)
	}
}

func TestWriteSetsPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "perms.txt")
	data := []byte("check perms")

	if err := Write(path, data, 0o640); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("unexpected permissions: got %o, want %o", info.Mode().Perm(), 0o640)
	}
}

func TestWriteOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	data := []byte("new content")
	if err := Write(path, data, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content mismatch: got %q, want %q", got, data)
	}
}

func TestWriteCreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "deep.txt")
	data := []byte("deep file")

	if err := Write(path, data, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("parent dir stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("parent is not a directory")
	}
}

func TestWriteAtomicity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.txt")
	data := []byte("atomic content")

	if err := Write(path, data, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Ensure no temporary files are left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == "atomic.txt" {
			continue
		}
		if filepath.Ext(entry.Name()) == ".tmp-" || len(entry.Name()) > len("atomic.txt") {
			t.Fatalf("temporary file left behind: %s", entry.Name())
		}
	}
}
