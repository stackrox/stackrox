package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// fakeBlobServer serves one in-memory blob with Azure-style ranged GET
// semantics: 206 + Content-Range for range requests, 200 for plain GETs.
func fakeBlobServer(data []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rg := r.Header.Get("x-ms-range")
		if rg == "" {
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
		// "bytes=lo-hi"
		var lo, hi int64
		if _, err := fmt.Sscanf(rg, "bytes=%d-%d", &lo, &hi); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if hi >= int64(len(data)) {
			hi = int64(len(data)) - 1
		}
		if lo > hi {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", lo, hi, len(data)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(data[lo : hi+1])
	}))
}

// Regression test for the range-assembly bug: when a blob is larger than
// rangeGetMaxParallel * rangeGetThreshold (16 * 512KB = 8MB), parts get
// resized bigger than the initial discovery fetch, and part 0 used to keep
// only the 512KB discovery read — silently dropping (partSize - 512KB)
// bytes. Observed in CI: 65MB vendor tarball fetched as 61.7MB.
func TestGetRangedLargeBlob(t *testing.T) {
	t.Parallel()
	// Exact size from the CI incident (65,292,470 bytes) — above the 8MB
	// cap threshold, exercising part resizing.
	data := make([]byte, 65_292_470)
	rnd := rand.New(rand.NewSource(42))
	rnd.Read(data)

	srv := fakeBlobServer(data)
	defer srv.Close()

	b := &AzureBackend{
		container: "c",
		baseURL:   srv.URL,
		client:    &http.Client{},
	}
	got, err := b.getRanged("vendor-cache-test", "fake-token")
	if err != nil {
		t.Fatalf("getRanged: %v", err)
	}
	if got == nil {
		t.Fatal("getRanged returned nil (silent miss)")
	}
	if len(got) != len(data) {
		t.Fatalf("size mismatch: got %d, want %d", len(got), len(data))
	}
	for i := range data {
		if data[i] != got[i] {
			t.Fatalf("byte %d differs: got %d, want %d", i, got[i], data[i])
		}
	}
}

// Small blobs (below rangeGetThreshold) must still return whole in one shot.
func TestGetRangedSmallBlob(t *testing.T) {
	t.Parallel()
	data := []byte(strings.Repeat("small blob payload ", 100)) // ~1.9KB
	srv := fakeBlobServer(data)
	defer srv.Close()

	b := &AzureBackend{
		container: "c",
		baseURL:   srv.URL,
		client:    &http.Client{},
	}
	got, err := b.getRanged("small", "fake-token")
	if err != nil {
		t.Fatalf("getRanged: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("small blob mismatch: got %d bytes, want %d", len(got), len(data))
	}
}

// Medium blob: above threshold but below the 16-part cap — the
// no-resize path must assemble exactly.
func TestGetRangedMediumBlob(t *testing.T) {
	t.Parallel()
	// ~4MB: 8 parts of 512KB, no cap resizing.
	data := make([]byte, 4_000_123)
	rnd := rand.New(rand.NewSource(7))
	rnd.Read(data)
	srv := fakeBlobServer(data)
	defer srv.Close()

	b := &AzureBackend{
		container: "c",
		baseURL:   srv.URL,
		client:    &http.Client{},
	}
	got, err := b.getRanged("medium", "fake-token")
	if err != nil {
		t.Fatalf("getRanged: %v", err)
	}
	if len(got) != len(data) {
		t.Fatalf("size mismatch: got %d, want %d", len(got), len(data))
	}
}
