package gocache

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

var _ = io.EOF

func TestTimeoutTrackerBounds(t *testing.T) {
	tt := NewTimeoutTracker()
	if got := tt.Timeout(); got != 3*time.Second {
		t.Fatalf("default timeout: %v", got)
	}
	for i := 0; i < 20; i++ {
		tt.Record(100 * time.Millisecond)
	}
	if got := tt.Timeout(); got != 500*time.Millisecond {
		t.Fatalf("floor: %v", got) // p95=100ms*4=400ms → floor 500ms
	}
	for i := 0; i < 20; i++ {
		tt.Record(5 * time.Second)
	}
	if got := tt.Timeout(); got != 15*time.Second {
		t.Fatalf("cap: %v", got) // p95=5s*4=20s → cap 15s
	}
}

func TestNegativesLifecycle(t *testing.T) {
	dir := t.TempDir()
	n := loadNegatives(dir)
	n.Add("abc")
	if !n.Has("abc") {
		t.Fatal("added negative not found")
	}
	n.Persist()

	n2 := loadNegatives(dir)
	if !n2.Has("abc") {
		t.Fatal("negative lost across persist/load")
	}
	n2.Clear("abc")
	if n2.Has("abc") {
		t.Fatal("cleared negative still present")
	}
	n2.Persist()
	n3 := loadNegatives(dir)
	if n3.Has("abc") {
		t.Fatal("cleared negative persisted")
	}
}

func TestNegativesBounded(t *testing.T) {
	n := loadNegatives(t.TempDir())
	for i := 0; i < negativesMax+100; i++ {
		n.Add(string(rune(i)) + "-id")
	}
	if len(n.set) > negativesMax {
		t.Fatalf("negatives exceeded cap: %d", len(n.set))
	}
}

func TestStreamToTempFileHash(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "x")
	tmp, hash, size, err := streamToTempFile(strings.NewReader("hello world"), dest)
	if err != nil || size != 11 {
		t.Fatalf("stream: %v %d", err, size)
	}
	data, _ := os.ReadFile(tmp)
	if string(data) != "hello world" {
		t.Fatalf("content: %q", data)
	}
	// hash is the sha256 of "hello world"
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != want {
		t.Fatalf("hash: %s", hash)
	}
	os.Remove(tmp)
}

func TestIngestDiskPathStreaming(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "go-output")
	os.WriteFile(src, []byte("package archive bytes"), 0o644)

	// Case 1: outputID matches content hash.
	sum := sha256Hex("package archive bytes")
	gotID, size := ingestDiskPathStreaming(dir, sum, src)
	if gotID != sum || size != 21 {
		t.Fatalf("match case: %q %d", gotID, size)
	}
	if _, err := os.Stat(filepath.Join(dir, sum[:2], sum+"-d")); err != nil {
		t.Fatal("data file not placed in standard layout")
	}

	// Case 2: go's OutputID disagrees — must store under content hash.
	wrong := sha256Hex("something else")
	gotID, size = ingestDiskPathStreaming(dir, wrong, src)
	if gotID != sum {
		t.Fatalf("mismatch case: stored under %q, want content hash", gotID)
	}
}
