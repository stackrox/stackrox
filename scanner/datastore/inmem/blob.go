package inmem

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore/libvuln/driver"
	"golang.org/x/sys/unix"
)

const (
	displayBlobPattern = "inmem-display-*.blob"
	enrichBlobPattern  = "inmem-enrich-*.blob"
)

// zstd encoder/decoder are safe for concurrent use by multiple goroutines, so a
// single package-level pair is shared across all fragments.
var (
	blobEnc *zstd.Encoder
	blobDec *zstd.Decoder
)

func init() {
	var err error
	blobEnc, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		panic(fmt.Sprintf("inmem: initializing zstd encoder: %v", err))
	}
	blobDec, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	if err != nil {
		panic(fmt.Sprintf("inmem: initializing zstd decoder: %v", err))
	}
}

// cleanStaleBlobs removes leftover display/enrichment blob files from dir (e.g.,
// after an unclean process termination).
func cleanStaleBlobs(dir string) {
	for _, pattern := range []string{displayBlobPattern, enrichBlobPattern, colsFilePattern} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			continue
		}
		for _, p := range matches {
			_ = os.Remove(p)
		}
	}
}

// mmapRegion owns a memory-mapped, read-only blob file. It is unmapped and the
// backing file removed either deterministically via release (driven by fragment
// reference counting) or automatically if the owning store becomes unreachable.
type mmapRegion struct {
	data    []byte
	path    string
	cleanup runtime.Cleanup
	once    sync.Once
}

// release unmaps the region and removes its file. Safe to call multiple times
// and concurrently with GC-scheduled cleanup.
func (r *mmapRegion) release() {
	r.once.Do(func() {
		r.cleanup.Stop()
		freeMmap(r.data, r.path)
	})
}

// freeMmap performs the actual unmap+remove. It captures only the raw slice and
// path so it can run as a GC cleanup without retaining the owning store.
func freeMmap(data []byte, path string) {
	if data != nil {
		_ = unix.Munmap(data)
	}
	if path != "" {
		_ = os.Remove(path)
	}
}

// attachCleanup registers a GC cleanup as a safety net in case a Store is
// abandoned without Close. Normal operation releases deterministically via
// reference counting.
func attachCleanup[T any](r *mmapRegion, obj *T) {
	data, path := r.data, r.path
	r.cleanup = runtime.AddCleanup(obj, func(struct{}) { freeMmap(data, path) }, struct{}{})
}

// displayStore holds the cold display fields (description, links, raw severity
// vector) off the Go heap.
//
// Each distinct string is zstd-compressed and appended to a single memory-mapped
// file once. The heap retains only per-dictionary-entry offset/length slices.
type displayStore struct {
	region *mmapRegion

	descOff  []int64
	descLen  []int32
	linkOff  []int64
	linkLen  []int32
	sevOff   []int64
	sevLen   []int32
	fileSize int64
}

func (d *displayStore) decode(off []int64, ln []int32, i int32) string {
	if d == nil || i < 0 || int(i) >= len(off) {
		return ""
	}
	b, err := blobDec.DecodeAll(d.region.data[off[i]:off[i]+int64(ln[i])], nil)
	if err != nil {
		return ""
	}
	return string(b)
}

func (d *displayStore) description(i int32) string { return d.decode(d.descOff, d.descLen, i) }
func (d *displayStore) links(i int32) string       { return d.decode(d.linkOff, d.linkLen, i) }
func (d *displayStore) severity(i int32) string    { return d.decode(d.sevOff, d.sevLen, i) }

func (d *displayStore) offHeapBytes() int64 {
	if d == nil {
		return 0
	}
	return d.fileSize
}

func (d *displayStore) release() {
	if d != nil && d.region != nil {
		d.region.release()
	}
}

// displayBuilder deduplicates and compresses display strings into a file as a
// fragment is built.
type displayBuilder struct {
	f    *os.File
	path string
	off  int64

	descIx *intern[string]
	linkIx *intern[string]
	sevIx  *intern[string]

	descOff, linkOff, sevOff []int64
	descLen, linkLen, sevLen []int32
}

func newDisplayBuilder(dir string) (*displayBuilder, error) {
	f, err := os.CreateTemp(dir, displayBlobPattern)
	if err != nil {
		return nil, fmt.Errorf("inmem: creating display blob: %w", err)
	}
	return &displayBuilder{
		f:      f,
		path:   f.Name(),
		descIx: newIntern[string](),
		linkIx: newIntern[string](),
		sevIx:  newIntern[string](),
	}, nil
}

// abort closes and deletes the backing file when a build fails before finish.
func (b *displayBuilder) abort() {
	if b.f != nil {
		_ = b.f.Close()
		b.f = nil
	}
	if b.path != "" {
		_ = os.Remove(b.path)
		b.path = ""
	}
}

func (b *displayBuilder) write(desc, links, sev string) (di, li, si int32, err error) {
	if di, err = b.appendStr(b.descIx, &b.descOff, &b.descLen, desc); err != nil {
		return 0, 0, 0, err
	}
	if li, err = b.appendStr(b.linkIx, &b.linkOff, &b.linkLen, links); err != nil {
		return 0, 0, 0, err
	}
	if si, err = b.appendStr(b.sevIx, &b.sevOff, &b.sevLen, sev); err != nil {
		return 0, 0, 0, err
	}
	return di, li, si, nil
}

func (b *displayBuilder) appendStr(ix *intern[string], offs *[]int64, lens *[]int32, s string) (int32, error) {
	id, isNew := ix.id(s)
	if !isNew {
		return id, nil
	}
	enc := blobEnc.EncodeAll([]byte(s), nil)
	if _, err := b.f.Write(enc); err != nil {
		return 0, fmt.Errorf("inmem: writing display blob: %w", err)
	}
	*offs = append(*offs, b.off)
	*lens = append(*lens, int32(len(enc)))
	b.off += int64(len(enc))
	return id, nil
}

func (b *displayBuilder) finish() (*displayStore, error) {
	err := b.f.Close()
	b.f = nil
	if err != nil {
		b.abort()
		return nil, fmt.Errorf("inmem: closing display blob: %w", err)
	}
	region, err := mapRegion(b.path, b.off)
	if err != nil {
		b.abort()
		return nil, err
	}
	b.path = ""
	d := &displayStore{
		region:   region,
		descOff:  b.descOff,
		descLen:  b.descLen,
		linkOff:  b.linkOff,
		linkLen:  b.linkLen,
		sevOff:   b.sevOff,
		sevLen:   b.sevLen,
		fileSize: b.off,
	}
	attachCleanup(region, d)
	return d, nil
}

type enrichRec struct {
	tags []string
	off  int64
	ln   int32
}

// enrichStore holds enrichment payloads off-heap and mirrors the Postgres
// GetEnrichment semantics: records for a given updater (kind) whose tags overlap
// the query tags, latest update operation, LIMIT 1.
//
// Lifetime is governed by atomic reference counting so an enrichStore shared
// across liveSets is unmapped only after the Store and every in-flight reader
// release it.
type enrichStore struct {
	refs   atomic.Int64
	region *mmapRegion

	recs     map[string][]enrichRec
	tagIdx   map[string]map[string]int32
	fileSize int64
}

func (e *enrichStore) retain() {
	if e != nil {
		e.refs.Add(1)
	}
}

func (e *enrichStore) release() {
	if e != nil && e.refs.Add(-1) == 0 && e.region != nil {
		e.region.release()
	}
}

func (e *enrichStore) GetEnrichment(kind string, tags []string) ([]driver.EnrichmentRecord, error) {
	if e == nil {
		return nil, nil
	}
	ti := e.tagIdx[kind]
	if ti == nil {
		return nil, nil
	}
	for _, tag := range tags {
		idx, ok := ti[tag]
		if !ok {
			continue
		}
		return []driver.EnrichmentRecord{decodeEnrichRecord(e, kind, int(idx))}, nil
	}
	return nil, nil
}

func decodeEnrichRecord(e *enrichStore, kind string, i int) driver.EnrichmentRecord {
	rec := e.recs[kind][i]
	b, err := blobDec.DecodeAll(e.region.data[rec.off:rec.off+int64(rec.ln)], nil)
	if err != nil {
		return driver.EnrichmentRecord{Tags: rec.tags}
	}
	return driver.EnrichmentRecord{Tags: rec.tags, Enrichment: b}
}

func (e *enrichStore) offHeapBytes() int64 {
	if e == nil {
		return 0
	}
	return e.fileSize
}

type enrichBuilder struct {
	f    *os.File
	path string
	off  int64

	recs   map[string][]enrichRec
	tagIdx map[string]map[string]int32
}

func newEnrichBuilder(dir string) (*enrichBuilder, error) {
	f, err := os.CreateTemp(dir, enrichBlobPattern)
	if err != nil {
		return nil, fmt.Errorf("inmem: creating enrichment blob: %w", err)
	}
	return &enrichBuilder{
		f:      f,
		path:   f.Name(),
		recs:   map[string][]enrichRec{},
		tagIdx: map[string]map[string]int32{},
	}, nil
}

func (b *enrichBuilder) abort() {
	if b.f != nil {
		_ = b.f.Close()
		b.f = nil
	}
	if b.path != "" {
		_ = os.Remove(b.path)
		b.path = ""
	}
}

func (b *enrichBuilder) add(kind string, tags []string, payload []byte) error {
	enc := blobEnc.EncodeAll(payload, nil)
	if _, err := b.f.Write(enc); err != nil {
		return fmt.Errorf("inmem: writing enrichment blob: %w", err)
	}
	rec := enrichRec{tags: tags, off: b.off, ln: int32(len(enc))}
	b.off += int64(len(enc))

	idx := int32(len(b.recs[kind]))
	b.recs[kind] = append(b.recs[kind], rec)
	ti := b.tagIdx[kind]
	if ti == nil {
		ti = map[string]int32{}
		b.tagIdx[kind] = ti
	}
	for _, tag := range tags {
		if _, exists := ti[tag]; !exists {
			ti[tag] = idx
		}
	}
	return nil
}

func (b *enrichBuilder) finish() (*enrichStore, error) {
	err := b.f.Close()
	b.f = nil
	if err != nil {
		b.abort()
		return nil, fmt.Errorf("inmem: closing enrichment blob: %w", err)
	}
	region, err := mapRegion(b.path, b.off)
	if err != nil {
		b.abort()
		return nil, err
	}
	b.path = ""
	e := &enrichStore{
		region:   region,
		recs:     b.recs,
		tagIdx:   b.tagIdx,
		fileSize: b.off,
	}
	e.refs.Store(1)
	attachCleanup(region, e)
	return e, nil
}

// mapRegion memory-maps path read-only with MADV_RANDOM so single-row zstd
// decodes do not trigger kernel readahead into the container page cache.
func mapRegion(path string, size int64) (*mmapRegion, error) {
	if size == 0 {
		_ = os.Remove(path)
		return &mmapRegion{}, nil
	}
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("inmem: opening blob for mmap: %w", err)
	}
	data, mmapErr := unix.Mmap(int(fd.Fd()), 0, int(size), unix.PROT_READ, unix.MAP_SHARED)
	closeErr := fd.Close()
	if mmapErr != nil {
		return nil, fmt.Errorf("inmem: mmap blob: %w", mmapErr)
	}
	if closeErr != nil {
		_ = unix.Munmap(data)
		return nil, fmt.Errorf("inmem: closing mmap fd: %w", closeErr)
	}
	_ = unix.Madvise(data, unix.MADV_RANDOM)
	return &mmapRegion{data: data, path: path}, nil
}
