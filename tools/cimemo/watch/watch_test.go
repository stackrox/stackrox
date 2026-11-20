package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckReadSet(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("package util"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644)

	readSet := []string{"a.go", "b.go"}

	// Nothing changed since "now"
	now := time.Now().UnixNano()
	valid, changed := CheckReadSet(dir, readSet, now)
	if !valid {
		t.Fatalf("expected valid (no changes), got changed=%v", changed)
	}

	// Modify a read-set file → should be invalid
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main // changed"), 0o644)
	valid, changed = CheckReadSet(dir, readSet, now)
	if valid {
		t.Fatal("expected invalid (a.go changed), got valid")
	}
	if len(changed) != 1 || changed[0] != "a.go" {
		t.Fatalf("expected [a.go], got %v", changed)
	}

	// Modify a NON-read-set file (README) → should still be valid
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("changed readme"), 0o644)
	valid, _ = CheckReadSet(dir, readSet, now)
	// Note: a.go is still changed from the previous test, so this will be invalid.
	// Reset the baseline:
	now2 := time.Now().UnixNano()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("changed again"), 0o644)
	valid, _ = CheckReadSet(dir, readSet, now2)
	if !valid {
		t.Fatal("README change should not invalidate read-set")
	}
}

func TestCheckReadSetDeletedFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)
	readSet := []string{"a.go"}
	now := time.Now().UnixNano()

	os.Remove(filepath.Join(dir, "a.go"))
	valid, changed := CheckReadSet(dir, readSet, now)
	if valid {
		t.Fatal("deleted read-set file should invalidate")
	}
	if len(changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(changed))
	}
}

func TestMonitorDetectsChanges(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)

	readSet := []string{"a.go"}
	m, err := New(dir, readSet)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	m.Start()

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify a read-set file
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main // v2"), 0o644)

	// Give the event time to propagate
	time.Sleep(200 * time.Millisecond)

	if m.IsValid() {
		t.Fatal("expected invalid after modifying read-set file")
	}
	changed := m.Changed()
	if len(changed) != 1 {
		t.Fatalf("expected 1 changed file, got %d", len(changed))
	}
}

func TestMonitorIgnoresNonReadSetFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644)

	readSet := []string{"a.go"}
	m, err := New(dir, readSet)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	m.Start()
	time.Sleep(100 * time.Millisecond)

	// Modify a non-read-set file
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("changed"), 0o644)
	time.Sleep(200 * time.Millisecond)

	if !m.IsValid() {
		t.Fatal("README change should not invalidate read-set monitor")
	}
}

func TestFilterToReadSet(t *testing.T) {
	changed := []string{"README.md", "a.go", "b.go", "docs/x.md"}
	readSet := []string{"a.go", "b.go", "c.go"}

	filtered := FilterToReadSet(changed, readSet)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 (a.go, b.go), got %v", filtered)
	}
}
