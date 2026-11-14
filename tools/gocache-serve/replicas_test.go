package main

import (
	"os"
	"os/exec"
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

// The vendor-blob scenario: GZIPPED input files. The backend gzips at
// rest and Get returns DECOMPRESSED bytes — the marker must hash the
// LOGICAL content (post-decompression) on both sides, or every gzipped
// input mismatches its own marker (observed in CI: spurious vendor
// regeneration on every run).
func TestStoreAllFetchAnyGzipInput(t *testing.T) {
	t.Setenv("GOCACHE_SERVE_BACKEND", "dir")
	t.Setenv("GOCACHE_SERVE_DIR_STORE", t.TempDir())
	// A gzipped tar, exactly like the vendor step produces.
	src := filepath.Join(t.TempDir(), "vendor.tar.gz")
	cmd := exec.Command("tar", "-czf", src, "-C", t.TempDir(), ".")
	// create a file to tar
	os.WriteFile(filepath.Join(t.TempDir(), "somefile"), []byte("vendored content"), 0o644)
	if err := cmd.Run(); err != nil {
		t.Skipf("tar unavailable: %v", err)
	}
	if err := storeAll("gz-key", src); err != nil {
		t.Fatalf("storeAll: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "out")
	if err := fetchAny("gz-key", dst); err != nil {
		t.Fatalf("gzipped input must verify (marker hashes logical content): %v", err)
	}
}
