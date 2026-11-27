package inmem

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

// Compile-time assertions that the in-memory implementations satisfy Scanner's
// matcher datastore interfaces.
var (
	_ datastore.MatcherStore        = (*Store)(nil)
	_ postgres.MatcherStore         = (*Store)(nil)
	_ postgres.MatcherMetadataStore = (*MetadataStore)(nil)
)

// Options configures an in-memory matcher Store.
type Options struct {
	// BlobDir is the directory where off-heap column (.col) and display /
	// enrichment (.blob) files are written and memory-mapped. Defaults to
	// os.TempDir().
	BlobDir string
	// CleanBlobDir, when true, removes leftover inmem-*.blob and
	// inmem-cols-*.col files from BlobDir on store construction (for dedicated
	// directories after unclean restarts).
	CleanBlobDir bool
	// MaxRows, when positive, rejects an update if the resulting total live row
	// count across all fragments would exceed MaxRows, keeping the previous
	// liveSet intact.
	MaxRows int
	// MaxOffHeapBytes, when positive, rejects an update if the resulting total
	// off-heap mmap footprint across all fragments would exceed MaxOffHeapBytes,
	// keeping the previous liveSet intact.
	MaxOffHeapBytes int64
}

// liveSet is an immutable, reference-counted set of per-updater fragments.
type liveSet struct {
	refs   atomic.Int64
	vuln   []*snapshot
	enrich map[string]*enrichStore
	dists  []claircore.Distribution
	rows   int
}

func (ls *liveSet) retain() {
	ls.refs.Add(1)
}

func (ls *liveSet) release() {
	if ls.refs.Add(-1) == 0 {
		for _, f := range ls.vuln {
			f.release()
		}
		for _, e := range ls.enrich {
			e.release()
		}
	}
}

// Store is an in-memory, column-native postgres.MatcherStore.
//
// Each updater's vulnerabilities and enrichments are compiled on import into
// immutable per-updater fragments with hot columnar arrays (.col) and cold
// display/enrichment blobs (.blob) backed by read-only mmap files. Published
// liveSets are reference-counted so replaced fragments are unmapped and deleted
// immediately once in-flight queries finish, without blocking readers or racing
// on munmap.
type Store struct {
	opts       Options
	nextFragID atomic.Uint32

	// liveMu protects ONLY the live pointer read+retain (readers) and pointer
	// swap (writers). It is never held during matching or fragment building.
	// Lock ordering: mu -> liveMu.
	liveMu sync.RWMutex
	live   *liveSet

	// mu serializes fragment updates, operation bookkeeping, and Close.
	mu          sync.Mutex
	closed      bool
	vulnFrags   map[string]*snapshot
	enrichFrags map[string]*enrichStore
	ops         map[driver.UpdateKind]map[string]driver.UpdateOperation
}

// NewMatcherStore returns an empty store ready to receive updates.
func NewMatcherStore(opts Options) (*Store, error) {
	if opts.BlobDir == "" {
		opts.BlobDir = os.TempDir()
	}
	if err := os.MkdirAll(opts.BlobDir, 0o700); err != nil {
		return nil, fmt.Errorf("inmem: creating blob dir: %w", err)
	}
	if opts.CleanBlobDir {
		cleanStaleBlobs(opts.BlobDir)
	}
	return &Store{
		opts:        opts,
		vulnFrags:   map[string]*snapshot{},
		enrichFrags: map[string]*enrichStore{},
		ops: map[driver.UpdateKind]map[string]driver.UpdateOperation{
			driver.VulnerabilityKind: {},
			driver.EnrichmentKind:    {},
		},
	}, nil
}

// Stats describes the resource footprint of the currently live fragment set.
type Stats struct {
	Rows           int
	Fragments      int
	ColsOffHeap    int64
	DisplayOffHeap int64
	EnrichOffHeap  int64
}

// Stats reports footprint information for observability.
func (s *Store) Stats() Stats {
	var st Stats
	ls := s.acquireLive()
	if ls == nil {
		return st
	}
	defer ls.release()

	st.Fragments = len(ls.vuln)
	for _, f := range ls.vuln {
		st.Rows += f.nrows
		st.DisplayOffHeap += f.display.offHeapBytes()
		st.ColsOffHeap += f.cols.offHeapBytes()
	}
	for _, e := range ls.enrich {
		st.EnrichOffHeap += e.offHeapBytes()
	}
	return st
}

// Close detaches the live fragment set and releases the store's owner
// references. Any in-flight queries finish safely on their retained liveSet and
// unmap its files upon returning.
func (s *Store) Close() {
	var (
		oldLive   *liveSet
		oldVulns  map[string]*snapshot
		oldEnrich map[string]*enrichStore
	)
	concurrency.WithLock(&s.mu, func() {
		if s.closed {
			return
		}
		s.closed = true
		oldLive = concurrency.WithLock1(&s.liveMu, func() *liveSet {
			prev := s.live
			s.live = nil
			return prev
		})
		oldVulns = s.vulnFrags
		oldEnrich = s.enrichFrags
		s.vulnFrags = nil
		s.enrichFrags = nil
		recordLiveMetrics(nil)
	})

	if oldLive != nil {
		oldLive.release()
	}
	for _, f := range oldVulns {
		f.release()
	}
	for _, e := range oldEnrich {
		e.release()
	}
}

func (s *Store) acquireLive() *liveSet {
	return concurrency.WithRLock1(&s.liveMu, func() *liveSet {
		ls := s.live
		if ls != nil {
			ls.retain()
		}
		return ls
	})
}

// publishLocked builds a fresh liveSet from the current staged fragments in
// deterministic updater order, retains every included fragment, updates
// Prometheus gauges, and atomically swaps it in under liveMu.
// Caller must hold s.mu. Returns the previous liveSet (if any) for release.
func (s *Store) publishLocked() *liveSet {
	names := make([]string, 0, len(s.vulnFrags))
	for name := range s.vulnFrags {
		names = append(names, name)
	}
	slices.Sort(names)

	ls := &liveSet{
		vuln:   make([]*snapshot, 0, len(names)),
		enrich: make(map[string]*enrichStore, len(s.enrichFrags)),
	}
	distSeen := make(map[claircore.Distribution]struct{})
	for _, name := range names {
		f := s.vulnFrags[name]
		f.retain()
		ls.vuln = append(ls.vuln, f)
		ls.rows += f.nrows
		for _, d := range f.knownDists {
			if _, ok := distSeen[d]; ok {
				continue
			}
			distSeen[d] = struct{}{}
			ls.dists = append(ls.dists, d)
		}
	}
	for k, e := range s.enrichFrags {
		e.retain()
		ls.enrich[k] = e
	}
	ls.refs.Store(1)

	old := concurrency.WithLock1(&s.liveMu, func() *liveSet {
		prev := s.live
		s.live = ls
		return prev
	})
	recordLiveMetrics(ls)
	return old
}

// checkLimitsLocked validates that replacing updater/kind with candidate rows
// and off-heap bytes would not exceed configured circuit-breaker ceilings.
// Caller must hold s.mu.
func (s *Store) checkLimitsLocked(replaceVuln string, newRows int, replaceEnrich string, newOffHeap int64) error {
	if s.opts.MaxRows > 0 {
		totalRows := newRows
		for u, f := range s.vulnFrags {
			if u == replaceVuln {
				continue
			}
			totalRows += f.nrows
		}
		if totalRows > s.opts.MaxRows {
			return fmt.Errorf("inmem: total rows %d exceeds max_rows limit %d", totalRows, s.opts.MaxRows)
		}
	}
	if s.opts.MaxOffHeapBytes > 0 {
		totalBytes := newOffHeap
		for u, f := range s.vulnFrags {
			if u == replaceVuln {
				continue
			}
			totalBytes += f.cols.offHeapBytes() + f.display.offHeapBytes()
		}
		for k, e := range s.enrichFrags {
			if k == replaceEnrich {
				continue
			}
			totalBytes += e.offHeapBytes()
		}
		if totalBytes > s.opts.MaxOffHeapBytes {
			return fmt.Errorf("inmem: total off-heap bytes %d exceeds max_offheap_bytes limit %d", totalBytes, s.opts.MaxOffHeapBytes)
		}
	}
	return nil
}

// ---- read path ----

// Get implements datastore.Vulnerability. It fans out across every vulnerability
// fragment, evaluates opts.Vulnerable (when provided) before decompressing cold
// display fields, and unions results across multiple IndexRecords that share a
// Package.ID.
func (s *Store) Get(ctx context.Context, records []*claircore.IndexRecord, opts datastore.GetOpts) (map[string][]*claircore.Vulnerability, error) {
	ls := s.acquireLive()
	if ls != nil {
		defer ls.release()
	}

	out := make(map[string][]*claircore.Vulnerability, len(records))
	seenByPkg := make(map[string]map[string]struct{}, len(records))
	cache := newDecodeCache()

	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if rec == nil || rec.Package == nil || rec.Package.Name == "" {
			continue
		}
		rid := rec.Package.ID
		if _, exists := out[rid]; !exists {
			out[rid] = nil
		}
		if ls == nil {
			continue
		}
		pkgSeen := seenByPkg[rid]
		emit := func(v *claircore.Vulnerability) {
			if pkgSeen == nil {
				pkgSeen = make(map[string]struct{})
				seenByPkg[rid] = pkgSeen
			}
			if _, dup := pkgSeen[v.ID]; dup {
				return
			}
			pkgSeen[v.ID] = struct{}{}
			out[rid] = append(out[rid], v)
		}
		for _, frag := range ls.vuln {
			if err := frag.matchRecord(ctx, rec, opts, cache, emit); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// globalID encodes an immutable (fragment ID, row) pair into a globally unique
// decimal ID across the entire lifetime of a Store, preventing collisions when
// concurrent matchers within one libvuln.Scan observe adjacent liveSets.
func globalID(fragID uint32, row int32) string {
	return strconv.FormatUint(uint64(fragID)<<32|uint64(uint32(row)), 10)
}

// GetEnrichment implements datastore.Enrichment.
func (s *Store) GetEnrichment(_ context.Context, kind string, tags []string) ([]driver.EnrichmentRecord, error) {
	ls := s.acquireLive()
	if ls == nil {
		return nil, nil
	}
	defer ls.release()

	return ls.enrich[kind].GetEnrichment(tags)
}

// Distributions returns the deduplicated known distributions across all
// fragments, mirroring scanner/datastore/postgres/distributions.go (including
// RHEL and Hummingbird CPE-derived distributions).
func (s *Store) Distributions(_ context.Context) ([]claircore.Distribution, error) {
	ls := s.acquireLive()
	if ls == nil {
		return nil, nil
	}
	defer ls.release()

	return slices.Clone(ls.dists), nil
}

// Initialized reports whether at least one vulnerability row is live.
func (s *Store) Initialized(_ context.Context) (bool, error) {
	ls := s.acquireLive()
	if ls == nil {
		return false, nil
	}
	defer ls.release()

	return ls.rows > 0, nil
}

func distKey(d claircore.Distribution) string {
	return d.DID + "\x00" + d.Name + "\x00" + d.Version + "\x00" + d.VersionCodeName +
		"\x00" + d.VersionID + "\x00" + d.Arch + "\x00" + d.CPE.String() + "\x00" + d.PrettyName
}

// ---- write path (Updater) ----

// UpdateVulnerabilities compiles a full set of vulnerabilities for one updater
// into an immutable columnar fragment.
func (s *Store) UpdateVulnerabilities(ctx context.Context, updater string, fp driver.Fingerprint, vulns []*claircore.Vulnerability) (uuid.UUID, error) {
	return s.UpdateVulnerabilitiesIter(ctx, updater, fp, sliceIter(vulns))
}

// UpdateVulnerabilitiesIter compiles the streamed vulnerabilities for one
// updater into an immutable columnar fragment, atomically publishes the updated
// liveSet, and releases the store's reference to any replaced fragment.
func (s *Store) UpdateVulnerabilitiesIter(ctx context.Context, updater string, fp driver.Fingerprint, it datastore.VulnerabilityIter) (uuid.UUID, error) {
	start := time.Now()
	fragID := s.nextFragID.Add(1)
	frag, err := buildFragment(ctx, s.opts.BlobDir, updater, fragID, s.opts.MaxRows, it)
	if err != nil {
		return uuid.Nil, err
	}
	ref := uuid.New()

	var (
		oldLive  *liveSet
		oldFrag  *snapshot
		abortErr error
	)
	concurrency.WithLock(&s.mu, func() {
		if s.closed {
			abortErr = errors.New("inmem: store is closed")
			return
		}
		newOffHeap := frag.cols.offHeapBytes() + frag.display.offHeapBytes()
		if err := s.checkLimitsLocked(updater, frag.nrows, "", newOffHeap); err != nil {
			abortErr = err
			return
		}
		oldFrag = s.vulnFrags[updater]
		s.vulnFrags[updater] = frag
		s.ops[driver.VulnerabilityKind][updater] = driver.UpdateOperation{
			Ref: ref, Updater: updater, Fingerprint: fp, Kind: driver.VulnerabilityKind, Date: time.Now(),
		}
		oldLive = s.publishLocked()
	})
	if abortErr != nil {
		frag.release()
		return uuid.Nil, abortErr
	}

	if oldLive != nil {
		oldLive.release()
	}
	if oldFrag != nil {
		oldFrag.release()
	}
	inmemUpdateDuration.WithLabelValues(string(driver.VulnerabilityKind)).Observe(time.Since(start).Seconds())
	return ref, nil
}

// DeltaUpdateVulnerabilities is not supported; Scanner V4's bundle import uses
// the full iterator form.
func (s *Store) DeltaUpdateVulnerabilities(_ context.Context, _ string, _ driver.Fingerprint, _ []*claircore.Vulnerability, _ []string) (uuid.UUID, error) {
	return uuid.Nil, errors.New("inmem: delta updates are not supported")
}

// UpdateEnrichments compiles a full set of enrichment records for one updater.
func (s *Store) UpdateEnrichments(ctx context.Context, kind string, fp driver.Fingerprint, es []driver.EnrichmentRecord) (uuid.UUID, error) {
	return s.UpdateEnrichmentsIter(ctx, kind, fp, enrichSliceIter(es))
}

// UpdateEnrichmentsIter compiles the streamed enrichment records for one
// updater into an off-heap fragment, atomically publishes the updated liveSet,
// and releases the store's reference to any replaced fragment.
func (s *Store) UpdateEnrichmentsIter(ctx context.Context, kind string, fp driver.Fingerprint, it datastore.EnrichmentIter) (uuid.UUID, error) {
	start := time.Now()
	eb, err := newEnrichBuilder(s.opts.BlobDir)
	if err != nil {
		return uuid.Nil, err
	}
	var iterErr error
	it(func(e *driver.EnrichmentRecord, err error) bool {
		if ctxErr := ctx.Err(); ctxErr != nil {
			iterErr = ctxErr
			return false
		}
		if err != nil {
			iterErr = err
			return false
		}
		if e == nil {
			return true
		}
		if err := eb.add(e.Tags, e.Enrichment); err != nil {
			iterErr = err
			return false
		}
		return true
	})
	if iterErr != nil {
		eb.abort()
		return uuid.Nil, fmt.Errorf("inmem: iterating enrichments for %s: %w", kind, iterErr)
	}
	frag, err := eb.finish()
	if err != nil {
		eb.abort()
		return uuid.Nil, err
	}
	ref := uuid.New()

	var (
		oldLive  *liveSet
		oldFrag  *enrichStore
		abortErr error
	)
	concurrency.WithLock(&s.mu, func() {
		if s.closed {
			abortErr = errors.New("inmem: store is closed")
			return
		}
		if err := s.checkLimitsLocked("", 0, kind, frag.offHeapBytes()); err != nil {
			abortErr = err
			return
		}
		oldFrag = s.enrichFrags[kind]
		s.enrichFrags[kind] = frag
		s.ops[driver.EnrichmentKind][kind] = driver.UpdateOperation{
			Ref: ref, Updater: kind, Fingerprint: fp, Kind: driver.EnrichmentKind, Date: time.Now(),
		}
		oldLive = s.publishLocked()
	})
	if abortErr != nil {
		frag.release()
		return uuid.Nil, abortErr
	}

	if oldLive != nil {
		oldLive.release()
	}
	if oldFrag != nil {
		oldFrag.release()
	}
	inmemUpdateDuration.WithLabelValues(string(driver.EnrichmentKind)).Observe(time.Since(start).Seconds())
	return ref, nil
}

// ---- update-operation bookkeeping ----

func (s *Store) kindsFor(kind driver.UpdateKind) []driver.UpdateKind {
	if kind == "" {
		return []driver.UpdateKind{driver.VulnerabilityKind, driver.EnrichmentKind}
	}
	return []driver.UpdateKind{kind}
}

func (s *Store) GetUpdateOperations(_ context.Context, kind driver.UpdateKind, updaters ...string) (map[string][]driver.UpdateOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string][]driver.UpdateOperation{}
	for _, k := range s.kindsFor(kind) {
		m := s.ops[k]
		if len(updaters) == 0 {
			for u, op := range m {
				out[u] = append(out[u], op)
			}
			continue
		}
		for _, u := range updaters {
			if op, ok := m[u]; ok {
				out[u] = append(out[u], op)
			}
		}
	}
	return out, nil
}

func (s *Store) GetLatestUpdateRefs(_ context.Context, kind driver.UpdateKind) (map[string][]driver.UpdateOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string][]driver.UpdateOperation{}
	for _, k := range s.kindsFor(kind) {
		for u, op := range s.ops[k] {
			out[u] = append(out[u], op)
		}
	}
	return out, nil
}

func (s *Store) GetLatestUpdateRef(_ context.Context, kind driver.UpdateKind) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest driver.UpdateOperation
	for _, k := range s.kindsFor(kind) {
		for _, op := range s.ops[k] {
			if op.Date.After(latest.Date) {
				latest = op
			}
		}
	}
	return latest.Ref, nil
}

// DeleteUpdateOperations drops both bookkeeping and the corresponding staged
// fragments for the referenced operations, republishing the liveSet.
func (s *Store) DeleteUpdateOperations(_ context.Context, refs ...uuid.UUID) (int64, error) {
	if len(refs) == 0 {
		return 0, nil
	}
	drop := make(map[uuid.UUID]struct{}, len(refs))
	for _, r := range refs {
		drop[r] = struct{}{}
	}

	var (
		n          int64
		oldLive    *liveSet
		oldVulns   []*snapshot
		oldEnrichs []*enrichStore
	)

	concurrency.WithLock(&s.mu, func() {
		if s.closed {
			return
		}
		for u, op := range s.ops[driver.VulnerabilityKind] {
			if _, ok := drop[op.Ref]; ok {
				delete(s.ops[driver.VulnerabilityKind], u)
				if f, ok := s.vulnFrags[u]; ok {
					delete(s.vulnFrags, u)
					oldVulns = append(oldVulns, f)
				}
				n++
			}
		}
		for k, op := range s.ops[driver.EnrichmentKind] {
			if _, ok := drop[op.Ref]; ok {
				delete(s.ops[driver.EnrichmentKind], k)
				if e, ok := s.enrichFrags[k]; ok {
					delete(s.enrichFrags, k)
					oldEnrichs = append(oldEnrichs, e)
				}
				n++
			}
		}
		if n > 0 {
			oldLive = s.publishLocked()
		}
	})

	if oldLive != nil {
		oldLive.release()
	}
	for _, f := range oldVulns {
		f.release()
	}
	for _, e := range oldEnrichs {
		e.release()
	}
	return n, nil
}

// GetUpdateDiff is not supported; the in-memory store retains only the latest
// fragment per updater.
func (s *Store) GetUpdateDiff(_ context.Context, _, _ uuid.UUID) (*driver.UpdateDiff, error) {
	return nil, errors.New("inmem: update diffs are not supported")
}

// GC is a no-op: the store retains only the latest fragment per updater.
func (s *Store) GC(_ context.Context, _ int) (int64, error) { return 0, nil }

// RecordUpdaterStatus is a no-op for the in-memory store.
func (s *Store) RecordUpdaterStatus(_ context.Context, _ string, _ time.Time, _ driver.Fingerprint, _ error) error {
	return nil
}

// RecordUpdaterSetStatus is a no-op for the in-memory store.
func (s *Store) RecordUpdaterSetStatus(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func sliceIter(vulns []*claircore.Vulnerability) datastore.VulnerabilityIter {
	return func(yield func(*claircore.Vulnerability, error) bool) {
		for _, v := range vulns {
			if !yield(v, nil) {
				return
			}
		}
	}
}

func enrichSliceIter(es []driver.EnrichmentRecord) datastore.EnrichmentIter {
	return func(yield func(*driver.EnrichmentRecord, error) bool) {
		for i := range es {
			if !yield(&es[i], nil) {
				return
			}
		}
	}
}
