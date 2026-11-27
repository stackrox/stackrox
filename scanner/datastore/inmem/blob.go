package inmem

import (
	"bufio"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync/atomic"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore/libvuln/driver"
	"golang.org/x/sys/unix"
)

const (
	displayBlobPattern = "inmem-display-*.blob"
	enrichBlobPattern  = "inmem-enrich-*.blob"
	blobWriteBufSize   = 256 * 1024
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
		panic(fmt.Errorf("inmem: initializing zstd encoder: %w", err))
	}
	blobDec, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	if err != nil {
		panic(fmt.Errorf("inmem: initializing zstd decoder: %w", err))
	}
}

// cleanStaleBlobs removes leftover display, enrichment, and column files from
// dir (e.g., after an unclean process termination).
func cleanStaleBlobs(dir string) {
	for _, pattern := range []string{displayBlobPattern, enrichBlobPattern, colsFilePattern} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			slog.Warn("searching for stale inmem files", "dir", dir, "pattern", pattern, "reason", err)
			continue
		}
		for _, p := range matches {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				slog.Warn("removing stale inmem file", "file", p, "reason", err)
			}
		}
	}
}

// mmapHandle is shared by mmapRegion and the GC runtime.AddCleanup fallback so
// unix.Munmap and os.Remove execute at most once even if explicit release()
// races with GC finalization.
type mmapHandle struct {
	data  []byte
	path  string
	freed atomic.Bool
}

func (h *mmapHandle) free() {
	if h == nil || !h.freed.CompareAndSwap(false, true) {
		return
	}
	if h.data != nil {
		_ = unix.Munmap(h.data)
	}
	if h.path != "" {
		_ = os.Remove(h.path)
	}
}

// mmapRegion owns a memory-mapped, read-only blob or column file. It is
// unmapped and the backing file removed either deterministically via release
// (driven by fragment reference counting) or automatically if the owning store
// becomes unreachable.
type mmapRegion struct {
	handle  *mmapHandle
	cleanup runtime.Cleanup
}

func (r *mmapRegion) data() []byte {
	if r == nil || r.handle == nil {
		return nil
	}
	return r.handle.data
}

// release unmaps the region and removes its file. Safe to call multiple times
// and concurrently with GC-scheduled cleanup.
func (r *mmapRegion) release() {
	if r == nil {
		return
	}
	r.cleanup.Stop()
	r.handle.free()
}

// attachCleanup registers a GC cleanup as a safety net in case a Store is
// abandoned without Close. The cleanup captures only r.handle (never obj or r).
func attachCleanup[T any](r *mmapRegion, obj *T) {
	if r == nil || r.handle == nil {
		return
	}
	h := r.handle
	r.cleanup = runtime.AddCleanup(obj, func(h *mmapHandle) { h.free() }, h)
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

func (d *displayStore) decode(off []int64, ln []int32, i int32) (string, error) {
	if d == nil || i < 0 || int(i) >= len(off) {
		return "", nil
	}
	raw := d.region.data()
	b, err := blobDec.DecodeAll(raw[off[i]:off[i]+int64(ln[i])], nil)
	if err != nil {
		return "", fmt.Errorf("inmem: decoding display blob entry %d: %w", i, err)
	}
	return string(b), nil
}

func (d *displayStore) description(i int32) (string, error) { return d.decode(d.descOff, d.descLen, i) }
func (d *displayStore) links(i int32) (string, error)       { return d.decode(d.linkOff, d.linkLen, i) }
func (d *displayStore) severity(i int32) (string, error)    { return d.decode(d.sevOff, d.sevLen, i) }

func (d *displayStore) offHeapBytes() int64 {
	if d == nil {
		return 0
	}
	return d.fileSize
}

func (d *displayStore) release() {
	if d != nil && d.region != nil {
		d.region.release()
		runtime.KeepAlive(d)
	}
}

// displayBuilder deduplicates and compresses display strings into a buffered
// file as a fragment is built.
type displayBuilder struct {
	f        *os.File
	bw       *bufio.Writer
	path     string
	off      int64
	maxBytes int64

	descIx *intern[string]
	linkIx *intern[string]
	sevIx  *intern[string]

	descOff, linkOff, sevOff []int64
	descLen, linkLen, sevLen []int32
}

func newDisplayBuilder(dir string, maxBytes int64) (*displayBuilder, error) {
	f, err := os.CreateTemp(dir, displayBlobPattern)
	if err != nil {
		return nil, fmt.Errorf("inmem: creating display blob: %w", err)
	}
	return &displayBuilder{
		f:        f,
		bw:       bufio.NewWriterSize(f, blobWriteBufSize),
		path:     f.Name(),
		maxBytes: maxBytes,
		descIx:   newIntern[string](),
		linkIx:   newIntern[string](),
		sevIx:    newIntern[string](),
	}, nil
}

// abort closes and deletes the backing file when a build fails before finish.
func (b *displayBuilder) abort() {
	if b.f != nil {
		_ = b.f.Close()
		b.f = nil
		b.bw = nil
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
	if len(enc) > math.MaxInt32 {
		return 0, fmt.Errorf("inmem: compressed display string exceeds int32 (%d bytes)", len(enc))
	}
	if b.maxBytes > 0 && b.off+int64(len(enc)) > b.maxBytes {
		return 0, fmt.Errorf("inmem: display blob exceeds max_offheap_bytes limit (%d)", b.maxBytes)
	}
	if _, err := b.bw.Write(enc); err != nil {
		return 0, fmt.Errorf("inmem: writing display blob: %w", err)
	}
	*offs = append(*offs, b.off)
	*lens = append(*lens, int32(len(enc)))
	b.off += int64(len(enc))
	return id, nil
}

func (b *displayBuilder) finish() (*displayStore, error) {
	if err := b.bw.Flush(); err != nil {
		b.abort()
		return nil, fmt.Errorf("inmem: flushing display blob: %w", err)
	}
	err := b.f.Close()
	b.f = nil
	b.bw = nil
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

// enrichStore holds enrichment payloads off-heap for a single updater kind and
// mirrors Postgres GetEnrichment semantics: records whose tags overlap the
// query tags, latest update operation, LIMIT 1.
//
// Lifetime is governed by atomic reference counting so an enrichStore shared
// across liveSets is unmapped only after the Store and every in-flight reader
// release it.
type enrichStore struct {
	refs   atomic.Int64
	region *mmapRegion

	recs     []enrichRec
	tagIdx   map[string]int32
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
		runtime.KeepAlive(e)
	}
}

func (e *enrichStore) GetEnrichment(tags []string) ([]driver.EnrichmentRecord, error) {
	if e == nil || len(e.tagIdx) == 0 {
		return nil, nil
	}
	for _, tag := range tags {
		idx, ok := e.tagIdx[tag]
		if !ok {
			continue
		}
		rec, err := decodeEnrichRecord(e, int(idx))
		if err != nil {
			return nil, err
		}
		return []driver.EnrichmentRecord{rec}, nil
	}
	return nil, nil
}

func decodeEnrichRecord(e *enrichStore, i int) (driver.EnrichmentRecord, error) {
	rec := e.recs[i]
	raw := e.region.data()
	b, err := blobDec.DecodeAll(raw[rec.off:rec.off+int64(rec.ln)], nil)
	if err != nil {
		return driver.EnrichmentRecord{}, fmt.Errorf("inmem: decoding enrichment record %d: %w", i, err)
	}
	return driver.EnrichmentRecord{Tags: slices.Clone(rec.tags), Enrichment: b}, nil
}

func (e *enrichStore) offHeapBytes() int64 {
	if e == nil {
		return 0
	}
	return e.fileSize
}

type enrichBuilder struct {
	f        *os.File
	bw       *bufio.Writer
	path     string
	off      int64
	maxBytes int64

	recs   []enrichRec
	tagIdx map[string]int32
}

func newEnrichBuilder(dir string, maxBytes int64) (*enrichBuilder, error) {
	f, err := os.CreateTemp(dir, enrichBlobPattern)
	if err != nil {
		return nil, fmt.Errorf("inmem: creating enrichment blob: %w", err)
	}
	return &enrichBuilder{
		f:        f,
		bw:       bufio.NewWriterSize(f, blobWriteBufSize),
		path:     f.Name(),
		maxBytes: maxBytes,
		tagIdx:   map[string]int32{},
	}, nil
}

func (b *enrichBuilder) abort() {
	if b.f != nil {
		_ = b.f.Close()
		b.f = nil
		b.bw = nil
	}
	if b.path != "" {
		_ = os.Remove(b.path)
		b.path = ""
	}
}

func (b *enrichBuilder) add(tags []string, payload []byte) error {
	if len(b.recs) >= math.MaxInt32 {
		return fmt.Errorf("inmem: enrichment record count overflow (%d)", len(b.recs))
	}
	enc := blobEnc.EncodeAll(payload, nil)
	if len(enc) > math.MaxInt32 {
		return fmt.Errorf("inmem: compressed enrichment payload exceeds int32 (%d bytes)", len(enc))
	}
	if b.maxBytes > 0 && b.off+int64(len(enc)) > b.maxBytes {
		return fmt.Errorf("inmem: enrichment blob exceeds max_offheap_bytes limit (%d)", b.maxBytes)
	}
	if _, err := b.bw.Write(enc); err != nil {
		return fmt.Errorf("inmem: writing enrichment blob: %w", err)
	}
	clonedTags := slices.Clone(tags)
	rec := enrichRec{tags: clonedTags, off: b.off, ln: int32(len(enc))}
	b.off += int64(len(enc))

	idx := int32(len(b.recs))
	b.recs = append(b.recs, rec)
	for _, tag := range clonedTags {
		if _, exists := b.tagIdx[tag]; !exists {
			b.tagIdx[tag] = idx
		}
	}
	return nil
}

func (b *enrichBuilder) finish() (*enrichStore, error) {
	if err := b.bw.Flush(); err != nil {
		b.abort()
		return nil, fmt.Errorf("inmem: flushing enrichment blob: %w", err)
	}
	err := b.f.Close()
	b.f = nil
	b.bw = nil
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
	return &mmapRegion{handle: &mmapHandle{data: data, path: path}}, nil
}
