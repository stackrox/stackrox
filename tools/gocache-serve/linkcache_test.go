package main

// Tests for the link-output caching components: stub generation (ELF
// structure + buildID round-trip through go's own reader), the
// manifestHash key, and LocalCache's spill tier. The canary test for
// the vendored linkActionID is in linkkey_test.go.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- Stub generation tests ---

func TestStubRoundTripViaGoTool(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not in PATH")
	}
	bid := "KW_p01HAVPPjG7AJW0uS/a7NzcEtoCA1L50a0KHUj/uWtZZh9eSawphbSPNaJm/T-HPGhv1eZaj8_0y54Od"
	outfile := filepath.Join(t.TempDir(), "stub")
	if err := writeStub(bid, outfile); err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command("go", "tool", "buildid", outfile).Output()
	if err != nil {
		t.Fatalf("go tool buildid on stub: %v", err)
	}
	if strings.TrimSpace(string(got)) != bid {
		t.Fatalf("round-trip mismatch:\n wrote  %s\n read   %s", bid, strings.TrimSpace(string(got)))
	}
}

func TestStubIsTiny(t *testing.T) {
	bid := "aaaaaaaaaaaaaaaaaaaaaaaa/bbbbbbbbbbbbbbbbbbbb/cccccccccccccccccccc/dddddddddddddddddddd"
	outfile := filepath.Join(t.TempDir(), "stub")
	if err := writeStub(bid, outfile); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(outfile)
	if err != nil {
		t.Fatal(err)
	}
	// Expected: 120 (header+phdr) + 12 (note header) + 4 (name) + len(bid)
	expected := int64(120 + 12 + 4 + len(bid))
	if st.Size() != expected {
		t.Fatalf("stub size %d, expected %d", st.Size(), expected)
	}
	if st.Size() > 512 {
		t.Fatalf("stub too large: %d bytes (should be ~250)", st.Size())
	}
}

func TestStubRejectsEmptyBuildID(t *testing.T) {
	err := writeStub("", filepath.Join(t.TempDir(), "stub"))
	if err == nil {
		t.Fatal("expected error for empty buildID")
	}
}

func TestStubDifferentBuildIDsDifferentFiles(t *testing.T) {
	dir := t.TempDir()
	bid1 := "aaaaaaaaaaaaaaaaaaaaaaaa/bbbbbbbbbbbbbbbbbbbb/cccccccccccccccccccc/dddddddddddddddddddd"
	bid2 := "eeeeeeeeeeeeeeeeeeeeee/ffffffffffffffffffff/00000000000000000000/11111111111111111111"
	f1, f2 := filepath.Join(dir, "s1"), filepath.Join(dir, "s2")
	if err := writeStub(bid1, f1); err != nil {
		t.Fatal(err)
	}
	if err := writeStub(bid2, f2); err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile(f1)
	b2, _ := os.ReadFile(f2)
	if string(b1) == string(b2) {
		t.Fatal("different buildIDs produced identical stubs")
	}
}

func TestStubELFHeader(t *testing.T) {
	bid := "a/b/c/d"
	f := filepath.Join(t.TempDir(), "stub")
	if err := writeStub(bid, f); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	// Magic
	if string(b[0:4]) != "\x7fELF" {
		t.Fatalf("bad ELF magic: %x", b[0:4])
	}
	// Class=64, Data=LSB, Version=1
	if b[4] != 2 || b[5] != 1 || b[6] != 1 {
		t.Fatalf("bad ident: class=%d data=%d ver=%d", b[4], b[5], b[6])
	}
	// PT_NOTE at program header offset 64
	phType := uint32(b[64]) | uint32(b[65])<<8 | uint32(b[66])<<16 | uint32(b[67])<<24
	if phType != 4 { // PT_NOTE
		t.Fatalf("program header type %d, expected PT_NOTE(4)", phType)
	}
	// Note at offset 120: namesz=4, tag=4, name="Go\0\0"
	if b[120] != 4 || b[121] != 0 || b[122] != 0 || b[123] != 0 {
		t.Fatalf("bad note namesz: %x", b[120:124])
	}
	if b[128] != 4 || b[129] != 0 || b[130] != 0 || b[131] != 0 {
		t.Fatalf("bad note tag: %x", b[128:132])
	}
	if string(b[132:136]) != "Go\x00\x00" {
		t.Fatalf("bad note name: %q", b[132:136])
	}
}

// --- LinkKeyActionID encoding tests ---

func TestLinkKeyActionIDURLEncoding(t *testing.T) {
	// A sum containing bytes that differ between Std and URL base64
	sum := [32]byte{}
	for i := range sum {
		sum[i] = byte(0x3E + i%2) // 0x3E='>' → '+' in std, '-' in URL
	}
	got := LinkKeyActionID(sum)
	if strings.ContainsAny(got, "+/") {
		t.Fatalf("actionID contains std-encoding chars: %s", got)
	}
	if len(got) != 20 {
		t.Fatalf("actionID length %d, expected 20", len(got))
	}
}

// --- manifestHash tests ---

func TestManifestHashStable(t *testing.T) {
	a := []manifestEntry{{id: "abc"}, {id: "def"}, {id: "ghi"}}
	b := []manifestEntry{{id: "ghi"}, {id: "abc"}, {id: "def"}}
	ha, hb := manifestHash(a), manifestHash(b)
	if ha == "" || ha != hb {
		t.Fatalf("manifestHash not order-independent: %q vs %q", ha, hb)
	}
}

func TestManifestHashEmpty(t *testing.T) {
	if manifestHash(nil) != "" {
		t.Fatal("empty manifest should hash to empty string")
	}
}

func TestManifestHashDistinguishes(t *testing.T) {
	a := []manifestEntry{{id: "abc"}}
	b := []manifestEntry{{id: "abd"}}
	if manifestHash(a) == manifestHash(b) {
		t.Fatal("different manifests produced same hash")
	}
}

// --- LocalCache spill tier tests ---

func TestLocalCacheSpill(t *testing.T) {
	dir := t.TempDir()
	spillDir := t.TempDir()
	lc, err := NewLocalCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	lc.SetSpill(spillDir)

	// Write to primary
	path, err := lc.Put("testentry", []byte("meta"), []byte("body"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("expected primary path, got %s", path)
	}

	// Read back via resolve
	got := lc.resolve("testentry")
	if got != path {
		t.Fatalf("resolve mismatch: %s vs %s", got, path)
	}

	// Has
	if !lc.Has("testentry") {
		t.Fatal("Has should be true after Put")
	}
	if lc.Has("nonexistent") {
		t.Fatal("Has should be false for nonexistent")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "file")
	if err := writeFileAtomic(p, []byte("content")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "content" {
		t.Fatalf("content mismatch: %q", b)
	}
	// No temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}
