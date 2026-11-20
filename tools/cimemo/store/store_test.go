package store

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBlobRoundTrip(t *testing.T) {
	s := newTestStore(t)

	hash, _, err := s.PutBlob(bytes.NewReader([]byte("hello world")))
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetBlob(hash)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}

func TestBlobDedup(t *testing.T) {
	s := newTestStore(t)

	h1, _, _ := s.PutBlob(bytes.NewReader([]byte("same content")))
	h2, _, _ := s.PutBlob(bytes.NewReader([]byte("same content")))

	if h1 != h2 {
		t.Fatalf("same content should have same hash: %s != %s", h1, h2)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	s := newTestStore(t)

	m := &Manifest{
		Key:     "test-key",
		Command: "echo hello",
		Blobs: []BlobRef{
			{Hash: "abc123", Path: "output.txt", Size: 12},
		},
	}
	if err := s.PutManifest(m); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetManifest("test-key")
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "test-key" || len(got.Blobs) != 1 {
		t.Fatalf("unexpected manifest: %+v", got)
	}
	if got.Blobs[0].Path != "output.txt" {
		t.Fatalf("unexpected path: %s", got.Blobs[0].Path)
	}
}

func TestSnapshotAndRestore(t *testing.T) {
	s := newTestStore(t)

	// Create a directory with files
	srcDir := t.TempDir()
	os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644)
	os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("content2"), 0o644)
	os.WriteFile(filepath.Join(srcDir, ".hidden"), []byte("skip me"), 0o644)

	// Snapshot (should skip .hidden)
	m, err := s.SnapshotDir("test", srcDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Blobs) != 2 {
		t.Fatalf("expected 2 blobs (hidden excluded), got %d", len(m.Blobs))
	}

	// Restore into a new directory
	dstDir := t.TempDir()
	count, _, err := s.RestoreManifest(m, dstDir)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 restored, got %d", count)
	}

	// Verify content
	data, _ := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if string(data) != "content1" {
		t.Fatalf("file1: got %q", data)
	}
	data, _ = os.ReadFile(filepath.Join(dstDir, "sub", "file2.txt"))
	if string(data) != "content2" {
		t.Fatalf("file2: got %q", data)
	}
}

func TestSnapshotWithExclusion(t *testing.T) {
	s := newTestStore(t)

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "keep.txt"), []byte("keep"), 0o644)
	os.WriteFile(filepath.Join(srcDir, "skip.tmp"), []byte("skip"), 0o644)

	m, err := s.SnapshotDir("test", srcDir, []string{"*.tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Blobs) != 1 {
		t.Fatalf("expected 1 blob (skip.tmp excluded), got %d", len(m.Blobs))
	}
}

func TestGCRemovesOldManifests(t *testing.T) {
	s := newTestStore(t)

	// Store a manifest
	m := &Manifest{
		Key:      "old",
		Blobs:    []BlobRef{{Hash: "abc", Path: "x", Size: 10}},
	}
	s.PutManifest(m)

	// Make it old (backdate the file)
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(s.manifestPath("old"), old, old)

	// GC with 24h max age
	freed, removed, err := s.GC(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if removed < 1 {
		t.Fatal("expected at least 1 removal (the old manifest)")
	}
	if freed < 0 {
		t.Fatal("freed should be >= 0")
	}

	// Manifest should be gone
	if s.HasManifest("old") {
		t.Fatal("old manifest should have been removed")
	}
}

func TestDedupAcrossManifests(t *testing.T) {
	s := newTestStore(t)

	// Create two directories that share a file
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	shared := []byte("shared content")
	os.WriteFile(filepath.Join(dir1, "common.txt"), shared, 0o644)
	os.WriteFile(filepath.Join(dir1, "only1.txt"), []byte("dir1 only"), 0o644)
	os.WriteFile(filepath.Join(dir2, "common.txt"), shared, 0o644)
	os.WriteFile(filepath.Join(dir2, "only2.txt"), []byte("dir2 only"), 0o644)

	// Snapshot both
	m1, _ := s.SnapshotDir("state1", dir1, nil)
	m2, _ := s.SnapshotDir("state2", dir2, nil)

	// Find common.txt in both
	var h1, h2 string
	for _, b := range m1.Blobs {
		if b.Path == "common.txt" {
			h1 = b.Hash
		}
	}
	for _, b := range m2.Blobs {
		if b.Path == "common.txt" {
			h2 = b.Hash
		}
	}

	if h1 == "" || h2 == "" {
		t.Fatal("common.txt not found in manifests")
	}
	if h1 != h2 {
		t.Fatal("same content should have same hash across manifests")
	}
}
