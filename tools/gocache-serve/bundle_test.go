package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestRealPFBBundle(t *testing.T) {
	for _, f := range []string{
		"/tmp/gcl/store/pfb-testjob-r2-0",
		"/tmp/gcl/store/pfb-testjob-r2-2",
		"/tmp/gcl/store/pfb-testjob-r2-4",
	} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Skipf("no store file: %v", err)
		}
		// decompress and inspect the raw stream directly
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: gzip: %v", f, err)
		}
		raw, err := io.ReadAll(zr)
		if err != nil {
			t.Fatalf("%s: gunzip: %v", f, err)
		}
		r := bytes.NewReader(raw)
		var count int32
		binary.Read(r, binary.LittleEndian, &count)
		t.Logf("%s: file=%dB raw=%dB count=%d", f, len(data), len(raw), count)
		// walk entries with ReadFull, count until short
		n := 0
		for i := int32(0); i < count; i++ {
			var keyLen, bodyLen int32
			if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
				t.Logf("%s: entry %d keyLen read: %v (pos %d/%d)", f, i, err, r.Size()-int64(r.Len()), len(raw))
				break
			}
			key := make([]byte, keyLen)
			if _, err := io.ReadFull(r, key); err != nil {
				t.Logf("%s: entry %d key: %v (pos %d/%d)", f, i, err, r.Size()-int64(r.Len()), len(raw))
				break
			}
			if err := binary.Read(r, binary.LittleEndian, &bodyLen); err != nil {
				t.Logf("%s: entry %d bodyLen: %v (pos %d/%d)", f, i, err, r.Size()-int64(r.Len()), len(raw))
				break
			}
			body := make([]byte, bodyLen)
			if _, err := io.ReadFull(r, body); err != nil {
				t.Logf("%s: entry %d body(%dB): %v (pos %d/%d)", f, i, bodyLen, err, r.Size()-int64(r.Len()), len(raw))
				break
			}
			n++
		}
		t.Logf("%s: fully decoded %d/%d entries", f, n, count)
	}
}

func TestBundleRoundTrip(t *testing.T) {
	// 1201 entries with sizes spanning the observed distribution:
	// 15B aux records, mid-size objects, empty bodies.
	var entries []BundleEntry
	for i := 0; i < 1201; i++ {
		var size int
		switch i % 10 {
		case 0:
			size = 15
		case 1:
			size = 0
		default:
			size = 100 + i*7
		}
		body := bytes.Repeat([]byte{byte(i % 251)}, size)
		entries = append(entries, BundleEntry{
			Key:  fmt.Sprintf("%064x", i),
			Body: body,
		})
	}
	data := SerializeBundle(entries)
	got, err := DeserializeBundle(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if len(got) != len(entries) {
		t.Fatalf("round-trip lost entries: packed %d, got %d", len(entries), len(got))
	}
	for i := range entries {
		if got[i].Key != entries[i].Key || !bytes.Equal(got[i].Body, entries[i].Body) {
			t.Fatalf("entry %d mismatch", i)
		}
	}
}
