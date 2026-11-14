package main

import (
	"os"
	"path/filepath"
	"testing"
)

// storeAll/fetchAny round trip in dir mode (single replica, but the
// marker semantics are identical).
func TestStoreAllFetchAnyRoundTrip(t *testing.T) {
	t.Setenv("GOCACHE_SERVE_BACKEND", "dir")
	t.Setenv("GOCACHE_SERVE_DIR_STORE", t.TempDir())
	src := filepath.Join(t.TempDir(), "content")
	os.WriteFile(src, []byte("proven-good content"), 0o644)
	if err := storeAll("rt-key", src); err != nil {
		t.Fatalf("storeAll: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "out")
	if err := fetchAny("rt-key", dst); err != nil {
		t.Fatalf("fetchAny after storeAll: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "proven-good content" {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

// The poisoned-blob scenario: content exists but has NO marker (stored
// before marker support) — fetchAny must treat it as a miss.
func TestFetchAnyRejectsUnmarkedContent(t *testing.T) {
	t.Setenv("GOCACHE_SERVE_BACKEND", "dir")
	store := t.TempDir()
	t.Setenv("GOCACHE_SERVE_DIR_STORE", store)
	// Store content WITHOUT a marker (simulating a pre-marker blob).
	b := &DirBackend{root: store}
	b.Put("poison-key", []byte("incomplete tree"))
	dst := filepath.Join(t.TempDir(), "out")
	err := fetchAny("poison-key", dst)
	if err == nil {
		t.Fatal("unmarked content must be a miss — the Nov-12 vendor poisoning would repeat")
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		t.Fatal("untrusted content must not be written to the output file")
	}
}

// Content with a marker that does NOT match (corruption or tampering)
// must be a miss.
func TestFetchAnyRejectsHashMismatch(t *testing.T) {
	t.Setenv("GOCACHE_SERVE_BACKEND", "dir")
	store := t.TempDir()
	t.Setenv("GOCACHE_SERVE_DIR_STORE", store)
	b := &DirBackend{root: store}
	b.Put("tampered-key", []byte("actual content"))
	b.Put("tampered-key.sha256", []byte("0000000000000000000000000000000000000000000000000000000000000000"))
	dst := filepath.Join(t.TempDir(), "out")
	if err := fetchAny("tampered-key", dst); err == nil {
		t.Fatal("hash-mismatched content must be a miss")
	}
}
