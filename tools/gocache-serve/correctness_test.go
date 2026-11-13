package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// failingGetBackend wraps a Backend but fails Gets for keys with a
// given prefix, simulating transient storage errors (timeouts, auth
// blips) during manifest reads.
type failingGetBackend struct {
	inner    Backend
	failPrefix string
	failErr  error
}

func (f *failingGetBackend) Get(id string) ([]byte, error) {
	if strings.HasPrefix(id, f.failPrefix) {
		return nil, f.failErr
	}
	return f.inner.Get(id)
}
func (f *failingGetBackend) Put(id string, body []byte) error      { return f.inner.Put(id, body) }
func (f *failingGetBackend) PutIfNotExist(id string, body []byte) (bool, error) {
	return f.inner.PutIfNotExist(id, body)
}
func (f *failingGetBackend) ListKeys() ([]string, error) { return f.inner.ListKeys() }
func (f *failingGetBackend) ListPrefix(p string) ([]string, error) {
	return f.inner.ListPrefix(p)
}
func (f *failingGetBackend) Close() error { return f.inner.Close() }

// TestManifestUploadAbortsOnReadFailure is the regression test for the
// manifest-shrinkage bug: a process whose manifest download fails must
// NEVER write a fresh (smaller) manifest over the accumulated one —
// observed in CI as a 2429-entry manifest replaced by a single
// process's 812 entries after a transient read error.
func TestManifestUploadAbortsOnReadFailure(t *testing.T) {
	t.Setenv("GOCACHE_SERVE_RUN_ID", "run-A")
	store := t.TempDir()
	b := &DirBackend{root: store}

	// Seed a manifest with a large working set from run-A.
	m1 := NewManifest("testjob")
	for i := 0; i < 100; i++ {
		m1.Record(fmt.Sprintf("entry%03d", i), "meta")
	}
	if _, err := m1.Upload(b); err != nil {
		t.Fatalf("seed upload: %v", err)
	}

	// A new session in the SAME run tries to upload a tiny working set,
	// but the manifest read fails (transient error).
	failing := &failingGetBackend{inner: b, failPrefix: "manifest-", failErr: errors.New("transient GET failure")}
	m2 := NewManifest("testjob")
	m2.Record("only-entry", "meta")
	if _, err := m2.Upload(failing); err == nil {
		t.Fatal("upload must fail when the manifest read fails — a success here means the clobber bug is back")
	}

	// The stored manifest must be UNCHANGED: still the 100 entries.
	entries, runID, err := DownloadManifest(b, "testjob")
	if err != nil {
		t.Fatalf("verify read: %v", err)
	}
	if runID != "run-A" {
		t.Fatalf("runID changed: %q", runID)
	}
	if len(entries) != 100 {
		t.Fatalf("manifest shrank: %d entries (want 100) — the correctness bug", len(entries))
	}
}

// TestManifestUploadCleanMissWritesFresh: a clean 404 (absent manifest)
// must still allow a fresh write — the abort only applies to ERRORS.
func TestManifestUploadCleanMissWritesFresh(t *testing.T) {
	t.Setenv("GOCACHE_SERVE_RUN_ID", "run-B")
	store := t.TempDir()
	b := &DirBackend{root: store}

	m := NewManifest("freshjob")
	m.Record("e1", "meta")
	if _, err := m.Upload(b); err != nil {
		t.Fatalf("fresh upload on absent manifest must succeed: %v", err)
	}
	entries, _, err := DownloadManifest(b, "freshjob")
	if err != nil || len(entries) != 1 {
		t.Fatalf("fresh manifest: entries=%d err=%v", len(entries), err)
	}
}

// TestFallbackBackendReadThroughAndBackfill: a miss on the primary that
// hits the secondary must serve the secondary's bytes AND backfill the
// primary (regions converge on demand).
func TestFallbackBackendReadThroughAndBackfill(t *testing.T) {
	primary := &DirBackend{root: filepath.Join(t.TempDir(), "p")}
	secondary := &DirBackend{root: filepath.Join(t.TempDir(), "s")}

	// Entry exists only on the secondary (e.g., west built it; east never did).
	if err := secondary.Put("shared-entry", []byte("content-from-hub")); err != nil {
		t.Fatalf("secondary put: %v", err)
	}

	fb := &fallbackBackend{primary: primary, secondary: secondary}

	// Read-through: miss on primary, hit on secondary.
	got, err := fb.Get("shared-entry")
	if err != nil || got == nil {
		t.Fatalf("read-through failed: got=%v err=%v", got, err)
	}
	if string(got) != "content-from-hub" {
		t.Fatalf("wrong content: %q", got)
	}

	// Backfill: eventually the primary has it too (async, bounded).
	for i := 0; i < 100; i++ {
		if p, _ := primary.Get("shared-entry"); p != nil {
			break
		}
		time.Sleep(5 * time.Millisecond) // yield to the backfill goroutine
	}
	p, _ := primary.Get("shared-entry")
	if p == nil {
		t.Fatal("primary was never backfilled — regions won't converge")
	}
	if string(p) != "content-from-hub" {
		t.Fatalf("backfilled content wrong: %q", p)
	}

	// Genuine miss on both: nil, nil (clean miss → go recompiles).
	got, err = fb.Get("nonexistent")
	if err != nil || got != nil {
		t.Fatalf("expected clean miss, got=%v err=%v", got, err)
	}
}

// TestFallbackBackendPrimaryErrorSurfaces: a primary ERROR must not be
// masked by the fallback (auth issues must stay visible).
func TestFallbackBackendPrimaryErrorSurfaces(t *testing.T) {
	primary := &failingGetBackend{inner: &DirBackend{root: t.TempDir()}, failPrefix: "", failErr: errors.New("primary down")}
	secondary := &DirBackend{root: t.TempDir()}
	secondary.Put("x", []byte("data"))

	fb := &fallbackBackend{primary: primary, secondary: secondary}
	_, err := fb.Get("x")
	if err == nil {
		t.Fatal("primary error was masked by fallback — auth/network issues must surface")
	}
}

// TestManifestWithinRunMergeGrows: sequential same-run sessions merge
// additively (the property that makes multi-process jobs accumulate).
func TestManifestWithinRunMergeGrows(t *testing.T) {
	t.Setenv("GOCACHE_SERVE_RUN_ID", "run-C")
	store := t.TempDir()
	b := &DirBackend{root: store}

	m1 := NewManifest("growjob")
	for i := 0; i < 50; i++ {
		m1.Record(fmt.Sprintf("a%02d", i), "")
	}
	m1.Upload(b)

	m2 := NewManifest("growjob")
	for i := 0; i < 30; i++ {
		m2.Record(fmt.Sprintf("b%02d", i), "")
	}
	m2.Upload(b)

	entries, _, err := DownloadManifest(b, "growjob")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(entries) != 80 {
		t.Fatalf("merge not additive: %d entries (want 80)", len(entries))
	}
}

// Ensure unused-import style issues don't hide real usage.
