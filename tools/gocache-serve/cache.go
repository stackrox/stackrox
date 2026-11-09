package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// Local disk cache manager
type LocalCache struct {
	dir string
	mu  sync.RWMutex
}

func NewLocalCache(dir string) (*LocalCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &LocalCache{dir: dir}, nil
}

func (lc *LocalCache) path(actionID string) string {
	if len(actionID) < 2 {
		return filepath.Join(lc.dir, actionID)
	}
	return filepath.Join(lc.dir, actionID[:2], actionID[2:])
}

func (lc *LocalCache) metaPath(actionID string) string {
	return lc.path(actionID) + ".meta"
}

func (lc *LocalCache) Has(actionID string) bool {
	_, err := os.Stat(lc.path(actionID))
	return err == nil
}

// GetMeta returns just the entry's sidecar and size without reading the
// body — the GOCACHEPROG get response only carries the DiskPath; Go reads
// the file itself, so reading the body here would be a redundant full copy
// of every entry on every local hit.
func (lc *LocalCache) GetMeta(actionID string) (meta []byte, size int64, ok bool) {
	meta, _ = os.ReadFile(lc.metaPath(actionID))
	st, err := os.Stat(lc.path(actionID))
	if err != nil {
		return nil, 0, false
	}
	return meta, st.Size(), true
}

func (lc *LocalCache) Get(actionID string) ([]byte, []byte, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	body, err := os.ReadFile(lc.path(actionID))
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	outputID, _ := os.ReadFile(lc.metaPath(actionID))
	return body, outputID, nil
}

func (lc *LocalCache) Put(actionID string, outputID, body []byte) (string, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	p := lc.path(actionID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, body, 0o644); err != nil {
		return "", err
	}
	if len(outputID) > 0 {
		if err := os.WriteFile(lc.metaPath(actionID), outputID, 0o644); err != nil {
			return "", err
		}
	}
	return p, nil
}

// Manifest tracks which cache entries a job uses, in request order, along
// with each entry's stored meta (the OutputID sidecar) so prefetch can
// write complete local entries.
//
// Run-scoped semantics: sessions within one CI run MERGE (a run's manifest
// is the union of its go invocations), but a new run REPLACES the stored
// manifest with its own working set. This avoids dead-entry accumulation
// (content-addressed keys for changed source never expire on their own —
// measured as 5x prefetch over-fetch on cli) and matches the production
// model where only default-branch runs save manifests: each save is a
// fresh canonical snapshot of that run's working set, like a GHA cache
// save, but without the archive.
type Manifest struct {
	mu      sync.Mutex
	entries []manifestEntry
	jobName string
	runID   string // GOCACHE_SERVE_RUN_ID; empty = legacy merge-always
	// readOnly: if true, Upload() is a no-op. PR runs set this so they
	// benefit from the master manifest's prefetch without overwriting
	// the canonical working set with PR-specific entries (which produce
	// different ActionIDs due to different source content).
	readOnly bool
}

type manifestEntry struct {
	id   string
	meta string // hex of the stored .meta (OutputID); may be empty for legacy lines
}

// manifestRunKey is the first line of a stored manifest: "#run:{runID}".
func manifestRunKey(id string) string { return "#run:" + id }

func NewManifest(jobName string) *Manifest {
	return &Manifest{jobName: jobName, runID: os.Getenv("GOCACHE_SERVE_RUN_ID")}
}

func (m *Manifest) Record(actionID, meta string) {
	if m == nil || m.jobName == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, manifestEntry{id: actionID, meta: meta})
}

// Upload merges this session's entries into the stored manifest. Merge (not
// last-writer-wins) is essential: every `go` invocation spawns a separate
// gocache-serve process, and a trivial trailing command (`go version`-class)
// would otherwise clobber the real workload's working set with 2-41 entries,
// leaving prefetch nothing to bulk-download on warm runs. This session's
// entries come first (most recent working set = prefetch priority), then any
// older entries not seen this session. Capped to bound growth across
// branches whose ActionIDs diverge.
// maxManifestEntries bounds manifest size (merge growth cap).
const maxManifestEntries = 20000

// Upload writes the run's manifest and returns the final entry set (used
// by the prefetch-bundle packer). Within a run, sessions merge; a new run
// replaces the stored manifest with its own working set.
func (m *Manifest) Upload(backend Backend) ([]manifestEntry, error) {
	if m == nil || m.jobName == "" || m.readOnly {
		return nil, nil
	}
	m.mu.Lock()
	entries := make([]manifestEntry, len(m.entries))
	copy(entries, m.entries)
	m.mu.Unlock()
	if len(entries) == 0 {
		return nil, nil // never clobber with an empty session
	}

	var merged []manifestEntry
	seen := make(map[string]bool, len(entries))
	addAll := func(list []manifestEntry) {
		for _, e := range list {
			if !seen[e.id] {
				seen[e.id] = true
				merged = append(merged, e)
			}
		}
	}
	addAll(entries) // this session first: prefetch priority = use order

	existing, existingRun, err := DownloadManifest(backend, m.jobName)
	if err != nil {
		// unreadable manifest: treat as absent, write fresh
		existing = nil
	}
	// Merge only within the same run. A different (older) run's entries are
	// replaced: dead keys from changed source otherwise accumulate forever.
	if m.runID == "" || existingRun == m.runID {
		addAll(existing)
	}
	if len(merged) > maxManifestEntries {
		merged = merged[:maxManifestEntries]
	}
	var buf strings.Builder
	if m.runID != "" {
		buf.WriteString(manifestRunKey(m.runID) + "\n")
	}
	for _, e := range merged {
		buf.WriteString(e.id)
		if e.meta != "" {
			buf.WriteString(" " + e.meta)
		}
		buf.WriteString("\n")
	}
	return merged, backend.Put("manifest-"+m.jobName, []byte(buf.String()))
}

// DownloadManifest fetches and parses a stored manifest, returning its
// entries and the run ID that wrote it ("" for legacy/no header). Lines
// are "actionID [meta]"; "#run:..." headers are stripped.
func DownloadManifest(backend Backend, jobName string) ([]manifestEntry, string, error) {
	if jobName == "" {
		return nil, "", nil
	}
	data, err := backend.Get("manifest-" + jobName)
	if err != nil {
		return nil, "", err
	}
	if data == nil {
		return nil, "", nil
	}
	var result []manifestEntry
	runID := ""
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#run:") {
			runID = strings.TrimPrefix(line, "#run:")
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		e := manifestEntry{id: parts[0]}
		if len(parts) >= 2 {
			e.meta = parts[1]
		}
		result = append(result, e)
	}
	return result, runID, nil
}

// AsyncUploader queues puts and uploads to cloud in background.
// Upload latency matters: entries under an active lease unblock followers
// on other runners, so lease-held entries are prioritized and drops are
// counted loudly (a dropped lease-held upload stalls followers).
type AsyncUploader struct {
	backend   Backend
	ch        chan uploadJob
	pri       chan uploadJob // lease-held entries: served before ch
	wg        sync.WaitGroup
	done      chan struct{}
	flushOnce sync.Once
	stats     UploadStats

	// onUploaded, if set, is invoked after each successful cloud Put
	// (used to release advisory compilation leases).
	onUploaded func(actionID string)

	dropped    int64 // jobs dropped because both queues were full
	droppedPri int64 // ...of which were lease-held (stalls followers)
}

type uploadJob struct {
	actionID string
	body     []byte
}

type UploadStats struct {
	mu        sync.Mutex
	uploaded  int64
	failed    int64
	bytesSent int64
}

func NewAsyncUploader(backend Backend, workers int) *AsyncUploader {
	if workers <= 0 {
		workers = 4
	}
	au := &AsyncUploader{
		backend: backend,
		ch:      make(chan uploadJob, 8192),
		pri:     make(chan uploadJob, 8192),
		done:    make(chan struct{}),
	}
	for i := 0; i < workers; i++ {
		au.wg.Add(1)
		go au.worker()
	}
	return au
}

func (au *AsyncUploader) worker() {
	defer au.wg.Done()
	for {
		// Lease-held (priority) uploads are served first: followers on
		// other runners are actively waiting on these entries, so their
		// latency converts directly into cross-runner hit rate.
		var job uploadJob
		var ok bool
		select {
		case job = <-au.pri:
			ok = true
		default:
			select {
			case job = <-au.pri:
				ok = true
			case job = <-au.ch:
				ok = true
			case <-au.done:
				return
			}
		}
		if !ok {
			continue
		}
		if err := au.backend.Put(job.actionID, job.body); err != nil {
			au.stats.mu.Lock()
			au.stats.failed++
			au.stats.mu.Unlock()
		} else {
			au.stats.mu.Lock()
			au.stats.uploaded++
			au.stats.bytesSent += int64(len(job.body))
			au.stats.mu.Unlock()
			if au.onUploaded != nil {
				au.onUploaded(job.actionID) // release lease, best-effort
			}
		}
	}
}

// QueuePri enqueues a lease-held upload on the priority lane, falling back
// to the normal lane if full. Drops are data loss for the shared cache
// (the entry exists only on this runner's disk), so the last resort blocks
// briefly — the local disk write already satisfied Go, so the put response
// is not on the compile critical path.
func (au *AsyncUploader) QueuePri(actionID string, body []byte) {
	select {
	case au.pri <- uploadJob{actionID, body}:
		return
	default:
	}
	select {
	case au.ch <- uploadJob{actionID, body}:
		return
	default:
	}
	atomic.AddInt64(&au.droppedPri, 1)
	log.Printf("gocache-serve: WARNING both upload queues full, blocking on %s", shortID(actionID))
	au.ch <- uploadJob{actionID, body} // block rather than lose the entry
}

// QueueDepth reports pending upload jobs (both lanes).
func (au *AsyncUploader) QueueDepth() int {
	return len(au.ch) + len(au.pri)
}

// Stats returns current upload counters.
func (au *AsyncUploader) Stats() (uploaded, failed, dropped int64) {
	au.stats.mu.Lock()
	defer au.stats.mu.Unlock()
	return au.stats.uploaded, au.stats.failed, atomic.LoadInt64(&au.dropped)
}

func (au *AsyncUploader) Queue(actionID string, body []byte) {
	select {
	case au.ch <- uploadJob{actionID, body}:
	default:
		// Never drop: the entry exists only on this runner's disk, and a
		// lost upload permanently caps the cache's hit rate. Block instead
		// (the local disk write already satisfied Go; the queue drains at
		// ~4 workers x 50ms = 80/s).
		atomic.AddInt64(&au.dropped, 1)
		au.ch <- uploadJob{actionID, body}
	}
}

func (au *AsyncUploader) Flush() {
	au.flushOnce.Do(func() {
		close(au.done)
		au.wg.Wait()
		au.stats.mu.Lock()
		defer au.stats.mu.Unlock()
		log.Printf("gocache-serve: uploads: %d ok, %d failed, %d bytes",
			au.stats.uploaded, au.stats.failed, au.stats.bytesSent)
	})
}
