package main

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
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
		m1.Record(fmt.Sprintf("%064x", i), "meta")
	}
	if _, err := m1.Upload(b); err != nil {
		t.Fatalf("seed upload: %v", err)
	}

	// A new session in the SAME run tries to upload a tiny working set,
	// but the manifest read fails (transient error).
	failing := &failingGetBackend{inner: b, failPrefix: "manifest-", failErr: errors.New("transient GET failure")}
	m2 := NewManifest("testjob")
	m2.Record(fmt.Sprintf("%064x", 9999), "meta")
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
	m.Record(fmt.Sprintf("%064x", 1), "meta")
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
		m1.Record(fmt.Sprintf("%064x", 1000+i), "")
	}
	m1.Upload(b)

	m2 := NewManifest("growjob")
	for i := 0; i < 30; i++ {
		m2.Record(fmt.Sprintf("%064x", 2000+i), "")
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

// countingBackend wraps a Backend and counts Get calls (test observability
// for the circuit-breaker tests).
type countingBackend struct {
	inner Backend
	gets  int32
}

func (c *countingBackend) Get(id string) ([]byte, error) {
	atomic.AddInt32(&c.gets, 1)
	return c.inner.Get(id)
}
func (c *countingBackend) Put(id string, body []byte) error { return c.inner.Put(id, body) }
func (c *countingBackend) PutIfNotExist(id string, body []byte) (bool, error) {
	return c.inner.PutIfNotExist(id, body)
}
func (c *countingBackend) ListKeys() ([]string, error)          { return c.inner.ListKeys() }
func (c *countingBackend) ListPrefix(p string) ([]string, error) { return c.inner.ListPrefix(p) }
func (c *countingBackend) Close() error                         { return c.inner.Close() }

// TestFallbackCircuitBreakerOpens: when both stores miss (genuinely cold
// source state), the secondary is consulted at most fallbackBreakerLimit
// times — after that, each miss pays ONE round trip, not two (the unfused
// fallback cost operator-bundle +72s in CI).
func TestFallbackCircuitBreakerOpens(t *testing.T) {
	primary := &DirBackend{root: t.TempDir()}
	secondary := &countingBackend{inner: &DirBackend{root: t.TempDir()}}
	fb := &fallbackBackend{primary: primary, secondary: secondary}

	for i := 0; i < 50; i++ {
		got, err := fb.Get(fmt.Sprintf("%064x", i))
		if err != nil || got != nil {
			t.Fatalf("get %d: expected clean miss, got=%v err=%v", i, got, err)
		}
	}
	if n := atomic.LoadInt32(&secondary.gets); n != fallbackBreakerLimit {
		t.Fatalf("breaker did not open: secondary queried %d times (want exactly %d)", n, fallbackBreakerLimit)
	}
}

// TestFallbackCircuitBreakerResetsOnHit: a single secondary hit resets the
// consecutive-miss count, so a mixed workload (some shared entries, some
// cold) keeps the fallback active.
func TestFallbackCircuitBreakerResetsOnHit(t *testing.T) {
	primary := &DirBackend{root: t.TempDir()}
	secStore := &DirBackend{root: t.TempDir()}
	// One entry the secondary has: a hit that should reset the breaker.
	hotID := fmt.Sprintf("%064x", 0xdead)
	secStore.Put(hotID, []byte("shared"))
	secondary := &countingBackend{inner: secStore}
	fb := &fallbackBackend{primary: primary, secondary: secondary}

	// 19 misses (below threshold).
	for i := 1; i <= 19; i++ {
		fb.Get(fmt.Sprintf("%064x", i))
	}
	// A hit: resets the consecutive-miss count.
	if got, _ := fb.Get(hotID); got == nil {
		t.Fatal("expected read-through hit")
	}
	// 15 more misses (kept under the threshold so the breaker itself
	// doesn't open): with the reset, all 15 consult the secondary.
	// Without the reset, the count would have continued at 19 and the
	// breaker would open on the very next miss — querying only 1 more.
	for i := 100; i < 115; i++ {
		fb.Get(fmt.Sprintf("%064x", i))
	}
	want := int32(19 + 1 + 15)
	if n := atomic.LoadInt32(&secondary.gets); n != want {
		t.Fatalf("reset-on-hit broken: secondary queried %d times (want %d)", n, want)
	}
}

// TestManifestParsingSanitizesIDs: malformed manifest lines (the CI panic
// came from a 425-char "ID") must be filtered at parse time, valid lines
// kept, meta preserved.
func TestManifestParsingSanitizesIDs(t *testing.T) {
	valid := fmt.Sprintf("%064x", 42)
	validWithMeta := fmt.Sprintf("%064x", 43)
	long := strings.Repeat("a", 425)          // the observed crash trigger
	nonHex := "z" + strings.Repeat("0", 63)   // right length, wrong charset
	short := "abcd"                           // too short
	upper := strings.ToUpper(valid)           // uppercase not accepted
	content := "#run:r1\n" +
		valid + "\n" +
		validWithMeta + " " + strings.Repeat("b", 64) + "\n" +
		long + "\n" + nonHex + "\n" + short + "\n" + upper + "\n"
	b := &DirBackend{root: t.TempDir()}
	if err := b.Put("manifest-sanitize", []byte(content)); err != nil {
		t.Fatalf("store: %v", err)
	}
	entries, runID, err := DownloadManifest(b, "sanitize")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if runID != "r1" {
		t.Fatalf("runID: %q", runID)
	}
	if len(entries) != 2 {
		t.Fatalf("sanitizer kept garbage: %d entries (want 2)", len(entries))
	}
	if entries[0].id != valid || entries[1].id != validWithMeta {
		t.Fatalf("wrong entries kept: %+v", entries)
	}
	if entries[1].meta != strings.Repeat("b", 64) {
		t.Fatalf("meta lost: %q", entries[1].meta)
	}
}

// TestIsHexID: boundary table for the manifest ID sanitizer.
func TestIsHexID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"abcd", false},                     // too short
		{strings.Repeat("0", 31), false},    // just under
		{strings.Repeat("0", 32), true},     // lower bound
		{strings.Repeat("f", 64), true},     // typical ActionID
		{strings.Repeat("f", 128), true},    // upper bound
		{strings.Repeat("f", 129), false},   // just over
		{"Z" + strings.Repeat("f", 63), false}, // non-hex
		{strings.ToUpper(strings.Repeat("f", 64)), false},
		{strings.Repeat("g", 64), false},
	}
	for _, c := range cases {
		if got := isHexID(c.in); got != c.want {
			t.Errorf("isHexID(%q...) = %v, want %v", c.in[:min(8, len(c.in))], got, c.want)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestGetRangedMalformedURLIsMiss: a blob URL that fails to parse must
// return a clean miss, not panic on a nil request (the CI crash was a
// segfault in getRanged's discarded NewRequest error).
func TestGetRangedMalformedURLIsMiss(t *testing.T) {
	b := &AzureBackend{
		account:   "test",
		container: "c",
		baseURL:   "http://bad url with spaces", // NewRequest must fail
		client:    &http.Client{},
	}
	// Call getRanged directly (Get would need a real token first) —
	// this is the exact frame that crashed in CI on a nil request.
	got, err := b.getRanged("abcdef0123456789abcdef0123456789", "fake-token")
	if err != nil {
		t.Fatalf("malformed URL must be a miss, not an error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil (miss), got %d bytes", len(got))
	}
}
