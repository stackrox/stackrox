package main

// Size-gated per-job prefetch bundles (pfb).
//
// The measured put-body distribution is bimodal: p50 = 15 bytes (Go's aux
// records), p95 = ~500KB (compiled objects), max ~88MB (linked binaries).
// Small entries are RTT-bound (46ms cross-region per request for as little
// as 15 bytes); large entries are throughput-bound (fine as direct gets).
//
// A pfb packs ONLY the small entries (< pfbMaxEntry, default 50KB — the
// crossover where per-request RTT starts to amortize) of one job's run
// manifest into ~500KB bundles, written by the same runs that write the
// manifest (eventually: master only). Prefetch then turns thousands of
// tiny requests into a handful of parallel bundle GETs, while large
// entries stream as parallel direct gets in manifest (use) order.
//
// Job-scoping removes the cross-job bundle-alignment problem entirely:
// each bundle contains exactly the entries its job's manifest lists, so
// over-fetch is zero by construction. Run-scoping (same as manifests)
// keeps pfbs fresh: a new saving run replaces the pfb set with its own
// working set; superseded bundles are reclaimed by the storage lifecycle
// policy (usage-based GC).
//
// Env knobs: GOCACHE_SERVE_PFB=0 disables; GOCACHE_SERVE_PFB_MAX_ENTRY
// (default 51200 = 50KB); GOCACHE_SERVE_PFB_BUNDLE_BYTES (default 512000
// = 500KB).

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PFBIndex lists a job's current prefetch bundles.
type PFBIndex struct {
	Bundles []string `json:"bundles"`
	Entries int      `json:"entries"`
	Bytes   int64    `json:"bytes"`
	RunID   string   `json:"run"` // the run that packed this set
}

func pfbIndexKey(job string) string { return "pfb-index-" + job }

// pfbEnvs reads the knobs (called once at startup in main).
func pfbEnvs() (enabled bool, maxEntry int, bundleBytes int64) {
	if os.Getenv("GOCACHE_SERVE_PFB") == "0" {
		return false, 0, 0
	}
	maxEntry = 50 * 1024
	if v := envInt("GOCACHE_SERVE_PFB_MAX_ENTRY", 0); v > 0 {
		maxEntry = v
	}
	bundleBytes = 500 * 1024
	if v := envInt("GOCACHE_SERVE_PFB_BUNDLE_BYTES", 0); v > 0 {
		bundleBytes = int64(v)
	}
	return true, maxEntry, bundleBytes
}

// UploadPFBs packs the small entries of the run's manifest (bodies read
// from the local cache, which holds everything this run used) into
// prefetch bundles and publishes the index. Called by manifest-writing
// runs after Manifest.Upload.
func UploadPFBs(backend Backend, jobName, runID string, entries []manifestEntry,
	local *LocalCache, maxEntry int, bundleBytes int64) error {
	if jobName == "" || len(entries) == 0 {
		return nil
	}

	// Collect small entries in manifest order (use order = prefetch order).
	var pack []BundleEntry
	var total int64
	for _, ent := range entries {
		body, _, err := local.Get(ent.id)
		if err != nil || body == nil || len(body) > maxEntry {
			continue
		}
		pack = append(pack, BundleEntry{Key: ent.id, Body: body})
		total += int64(len(body))
	}
	if len(pack) == 0 {
		return nil
	}

	// Chunk into bundles and upload.
	idx := PFBIndex{RunID: runID}
	var cur []BundleEntry
	var curSize int64
	seq := 0
	flushBundle := func() error {
		if len(cur) == 0 {
			return nil
		}
		data := SerializeBundle(cur)
		name := fmt.Sprintf("pfb-%s-%d", sanitizeJob(jobName), seq)
		if runID != "" {
			name = fmt.Sprintf("pfb-%s-%s-%d", sanitizeJob(jobName), sanitizeJob(runID), seq)
		}
		if err := backend.Put(name, data); err != nil {
			return fmt.Errorf("pfb upload %s: %w", name, err)
		}
		idx.Bundles = append(idx.Bundles, name)
		idx.Entries += len(cur)
		idx.Bytes += curSize
		seq++
		cur, curSize = nil, 0
		return nil
	}
	for _, e := range pack {
		if curSize+int64(len(e.Body)) > bundleBytes && len(cur) > 0 {
			if err := flushBundle(); err != nil {
				return err
			}
		}
		cur = append(cur, e)
		curSize += int64(len(e.Body))
	}
	if err := flushBundle(); err != nil {
		return err
	}
	idata, _ := json.Marshal(idx)
	if err := backend.Put(pfbIndexKey(jobName), idata); err != nil {
		return fmt.Errorf("pfb index upload: %w", err)
	}
	log.Printf("gocache-serve: pfb: packed %d small entries (%d bytes) into %d bundles",
		idx.Entries, idx.Bytes, len(idx.Bundles))
	return nil
}

// PrefetchPFBs downloads a job's prefetch bundles in parallel and unpacks
// their entries into the local cache. metas supplies the OutputID sidecars
// (from the manifest). Returns the number of entries unpacked.
func PrefetchPFBs(backend Backend, jobName string, local *LocalCache, metas map[string]string) int {
	if jobName == "" {
		return 0
	}
	data, err := backend.Get(pfbIndexKey(jobName))
	if err != nil || data == nil {
		return 0 // no pfb set yet (first run) — per-entry prefetch handles it
	}
	var idx PFBIndex
	if err := json.Unmarshal(data, &idx); err != nil || len(idx.Bundles) == 0 {
		return 0
	}
	start := time.Now()
	var mu sync.Mutex
	var unpacked int
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16)
	for _, b := range idx.Bundles {
		wg.Add(1)
		go func(bundleKey string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			blob, err := backend.Get(bundleKey)
			if err != nil || blob == nil {
				log.Printf("gocache-serve: pfb: bundle %s missing (%v)", bundleKey, err)
				return // stale bundle (replaced by a newer run) — skip
			}
			entries, err := DeserializeBundle(blob)
			if err != nil {
				log.Printf("gocache-serve: pfb: bundle %s decode failed: %v (len=%d)", bundleKey, err, len(blob))
				return
			}
			n := 0
			for _, e := range entries {
				local.Put(e.Key, []byte(metas[e.Key]), e.Body)
				n++
			}
			mu.Lock()
			unpacked += n
			mu.Unlock()
		}(b)
	}
	wg.Wait()
	log.Printf("gocache-serve: pfb: unpacked %d entries from %d bundles in %v",
		unpacked, len(idx.Bundles), time.Since(start).Round(time.Millisecond))
	return unpacked
}

func sanitizeJob(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// tokenDiskCache persists the Azure OAuth token across gocache-serve
// processes within a job (every `go` invocation spawns a new process;
// without this each pays a ~300ms token round trip).
type tokenDiskCache struct {
	path string
}

func newTokenDiskCache() *tokenDiskCache {
	dir := os.Getenv("GOCACHE_SERVE_DIR")
	if dir == "" {
		dir = "/tmp/gocache-serve"
	}
	return &tokenDiskCache{path: filepath.Join(dir, "token.json")}
}

func (t *tokenDiskCache) Load() (token string, exp time.Time, ok bool) {
	b, err := os.ReadFile(t.path)
	if err != nil {
		return "", time.Time{}, false
	}
	var v struct {
		Token string    `json:"token"`
		Exp   time.Time `json:"exp"`
	}
	if json.Unmarshal(b, &v) != nil || v.Token == "" {
		return "", time.Time{}, false
	}
	return v.Token, v.Exp, true
}

func (t *tokenDiskCache) Save(token string, exp time.Time) {
	os.MkdirAll(filepath.Dir(t.path), 0o755)
	b, _ := json.Marshal(struct {
		Token string    `json:"token"`
		Exp   time.Time `json:"exp"`
	}{token, exp})
	// best-effort: races between concurrent processes write identical content
	os.WriteFile(t.path, b, 0o600)
}
