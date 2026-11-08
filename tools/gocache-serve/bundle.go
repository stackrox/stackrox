package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// BundleEntry is a single cache entry within a bundle.
type BundleEntry struct {
	Key  string
	Body []byte
}

// SerializeBundle encodes entries into a gzip-compressed binary blob.
// Format: gzip([int32 count] then per entry: [int32 keyLen][key][int32 bodyLen][body])
// Go build cache entries (export data, DWARF, object code) compress ~2-3x.
func SerializeBundle(entries []BundleEntry) []byte {
	var raw bytes.Buffer
	binary.Write(&raw, binary.LittleEndian, int32(len(entries)))
	for _, e := range entries {
		binary.Write(&raw, binary.LittleEndian, int32(len(e.Key)))
		raw.WriteString(e.Key)
		binary.Write(&raw, binary.LittleEndian, int32(len(e.Body)))
		raw.Write(e.Body)
	}

	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	zw.Write(raw.Bytes())
	zw.Close()
	return compressed.Bytes()
}

// DeserializeBundle decodes a gzip-compressed bundle blob back into entries.
func DeserializeBundle(data []byte) ([]BundleEntry, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip open: %w", err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("gzip read: %w", err)
	}

	r := bytes.NewReader(raw)
	var count int32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("bundle count: %w", err)
	}
	entries := make([]BundleEntry, 0, count)
	for i := int32(0); i < count; i++ {
		var keyLen, bodyLen int32
		if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
			return nil, fmt.Errorf("key len %d: %w", i, err)
		}
		key := make([]byte, keyLen)
		if _, err := r.Read(key); err != nil {
			return nil, fmt.Errorf("key %d: %w", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &bodyLen); err != nil {
			return nil, fmt.Errorf("body len %d: %w", i, err)
		}
		body := make([]byte, bodyLen)
		if _, err := r.Read(body); err != nil {
			return nil, fmt.Errorf("body %d: %w", i, err)
		}
		entries = append(entries, BundleEntry{Key: string(key), Body: body})
	}
	return entries, nil
}

// BundledBackend wraps a raw cloud Backend with entry bundling:
//   - PUT: small entries are collected into bundles (~50 entries or ~5MB);
//     large entries go as individual blobs. One HTTP PUT per bundle instead
//     of one per entry.
//   - GET: checks individual blob first, then bundle index, then downloads
//     the entire bundle (getting ~50 entries in one HTTP GET). Parsed
//     bundles are cached in memory — subsequent requests for entries in the
//     same bundle are served without any cloud request.
//
// This reduces HTTP requests by up to 50x for both reads and writes.
type BundledBackend struct {
	inner Backend

	// Bundle index: actionID → bundle blob key
	indexMu sync.RWMutex
	index   map[string]string

	// Upload bundling state
	uploadMu       sync.Mutex
	pendingEntries []BundleEntry
	pendingSize    int64
	pendingBytes   int64 // total serialized size estimate
	maxEntries     int
	maxSize        int64
	bundleSeq      int64

	// Download bundle cache: bundle key → parsed entries
	cacheMu     sync.RWMutex
	bundleCache map[string][]BundleEntry

	// Stats
	bundlesUploaded int64
	bundlesDownload int64
	entriesSavedGet int64 // cloud GETs avoided by bundle cache
}

func NewBundledBackend(inner Backend, maxEntries int, maxSize int64) (*BundledBackend, error) {
	bb := &BundledBackend{
		inner:       inner,
		index:       make(map[string]string),
		maxEntries:  maxEntries,
		maxSize:     maxSize,
		bundleCache: make(map[string][]BundleEntry),
	}

	// Load existing bundle index from cloud
	indexData, err := inner.Get("bundle-index")
	if err != nil {
		return nil, fmt.Errorf("load bundle index: %w", err)
	}
	if indexData != nil {
		if err := json.Unmarshal(indexData, &bb.index); err != nil {
			log.Printf("gocache-serve: bundle index decode failed (starting fresh): %v", err)
		} else {
			log.Printf("gocache-serve: loaded bundle index: %d entries", len(bb.index))
		}
	}

	return bb, nil
}

func (bb *BundledBackend) Get(actionID string) ([]byte, error) {
	// Try individual blob first (large entries, or entries not yet bundled)
	body, err := bb.inner.Get(actionID)
	if err != nil {
		return nil, err
	}
	if body != nil {
		return body, nil
	}

	// Check bundle index
	bb.indexMu.RLock()
	bundleKey := bb.index[actionID]
	bb.indexMu.RUnlock()
	if bundleKey == "" {
		return nil, nil // not found anywhere
	}

	// Check if bundle is already downloaded and cached in memory
	bb.cacheMu.RLock()
	entries, cached := bb.bundleCache[bundleKey]
	bb.cacheMu.RUnlock()

	if !cached {
		// Download the entire bundle — one HTTP GET for ~50 entries
		atomic.AddInt64(&bb.bundlesDownload, 1)
		bundleData, err := bb.inner.Get(bundleKey)
		if err != nil {
			return nil, fmt.Errorf("bundle %s: %w", bundleKey, err)
		}
		if bundleData == nil {
			// Bundle was deleted or index is stale
			return nil, nil
		}
		entries, err = DeserializeBundle(bundleData)
		if err != nil {
			return nil, fmt.Errorf("bundle %s decode: %w", bundleKey, err)
		}

		// Cache the parsed bundle
		bb.cacheMu.Lock()
		bb.bundleCache[bundleKey] = entries
		bb.cacheMu.Unlock()
	} else {
		atomic.AddInt64(&bb.entriesSavedGet, 1)
	}

	// Find the requested entry
	for _, e := range entries {
		if e.Key == actionID {
			return e.Body, nil
		}
	}
	return nil, nil // in index but not in bundle (stale index)
}

func (bb *BundledBackend) Put(actionID string, body []byte) error {
	// Large entries go as individual blobs (no benefit from bundling)
	if int64(len(body)) > bb.maxSize {
		return bb.inner.Put(actionID, body)
	}

	bb.uploadMu.Lock()
	defer bb.uploadMu.Unlock()

	bb.pendingEntries = append(bb.pendingEntries, BundleEntry{Key: actionID, Body: body})
	bb.pendingSize += int64(len(body))

	if len(bb.pendingEntries) >= bb.maxEntries || bb.pendingSize >= bb.maxSize {
		return bb.flushBundle()
	}
	return nil
}

func (bb *BundledBackend) flushBundle() error {
	if len(bb.pendingEntries) == 0 {
		return nil
	}

	bundleKey := fmt.Sprintf("bundle-%d-%d", time.Now().Unix(), atomic.AddInt64(&bb.bundleSeq, 1))
	data := SerializeBundle(bb.pendingEntries)

	if err := bb.inner.Put(bundleKey, data); err != nil {
		// Fallback: upload entries individually if bundle fails
		log.Printf("gocache-serve: bundle upload failed, falling back to individual: %v", err)
		for _, e := range bb.pendingEntries {
			if err := bb.inner.Put(e.Key, e.Body); err != nil {
				return fmt.Errorf("individual fallback %s: %w", e.Key[:8], err)
			}
		}
		bb.pendingEntries = nil
		bb.pendingSize = 0
		return nil
	}

	atomic.AddInt64(&bb.bundlesUploaded, 1)

	// Update the in-memory index
	bb.indexMu.Lock()
	for _, e := range bb.pendingEntries {
		bb.index[e.Key] = bundleKey
	}
	bb.indexMu.Unlock()

	log.Printf("gocache-serve: bundled %d entries (%d bytes) into %s",
		len(bb.pendingEntries), bb.pendingSize, bundleKey)

	bb.pendingEntries = nil
	bb.pendingSize = 0
	return nil
}

func (bb *BundledBackend) ListKeys() ([]string, error) {
	return bb.inner.ListKeys()
}

func (bb *BundledBackend) Close() error {
	// Flush any pending bundle
	bb.uploadMu.Lock()
	bb.flushBundle()
	bb.uploadMu.Unlock()

	// Upload the updated bundle index
	bb.indexMu.RLock()
	indexData, _ := json.Marshal(bb.index)
	bb.indexMu.RUnlock()
	if err := bb.inner.Put("bundle-index", indexData); err != nil {
		return fmt.Errorf("upload bundle index: %w", err)
	}

	log.Printf("gocache-serve: bundles: %d uploaded, %d downloaded, %d cache-hits from bundles",
		atomic.LoadInt64(&bb.bundlesUploaded),
		atomic.LoadInt64(&bb.bundlesDownload),
		atomic.LoadInt64(&bb.entriesSavedGet))

	return bb.inner.Close()
}
