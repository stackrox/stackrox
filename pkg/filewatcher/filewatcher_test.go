package filewatcher

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileAppears(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	var called int
	handler := func(_ []byte) error { called++; return nil }
	w := New(filePath, 24*time.Hour, handler)

	w.check()
	assert.Equal(t, 0, called, "handler must not be called when file is absent")

	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0600))
	w.check()
	assert.Equal(t, 1, called, "handler must be called when file appears")
}

func TestFileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "nonexistent.json")

	var called int
	handler := func(_ []byte) error { called++; return nil }
	w := New(filePath, 24*time.Hour, handler)

	w.check()
	assert.Equal(t, 0, called)
	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)
}

func TestFileDeletedResetsHash(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	content := []byte("content")

	var called int
	handler := func(_ []byte) error { called++; return nil }
	w := New(filePath, 24*time.Hour, handler)

	require.NoError(t, os.WriteFile(filePath, content, 0600))
	w.check()
	assert.Equal(t, 1, called)
	assert.NotEqual(t, [sha256.Size]byte{}, w.lastHash)

	require.NoError(t, os.Remove(filePath))
	w.check()
	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)

	require.NoError(t, os.WriteFile(filePath, content, 0600))
	w.check()
	assert.Equal(t, 2, called, "handler must be called again after file deletion and re-creation")
}

func TestFileChanges(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	var received []string
	handler := func(data []byte) error { received = append(received, string(data)); return nil }
	w := New(filePath, 24*time.Hour, handler)

	require.NoError(t, os.WriteFile(filePath, []byte("v1"), 0600))
	w.check()

	require.NoError(t, os.WriteFile(filePath, []byte("v2"), 0600))
	w.check()

	assert.Equal(t, []string{"v1", "v2"}, received)
}

func TestFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0600))

	var called int
	handler := func(_ []byte) error { called++; return nil }
	w := New(filePath, 24*time.Hour, handler)

	w.check()
	w.check()
	assert.Equal(t, 1, called, "handler must not be called for unchanged content")
}

func TestHandlerErrorCausesRetry(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	content := []byte("content")
	require.NoError(t, os.WriteFile(filePath, content, 0600))

	var called int
	handler := func(_ []byte) error {
		called++
		if called == 1 {
			return errors.New("transient failure")
		}
		return nil
	}
	w := New(filePath, 24*time.Hour, handler)

	assert.Equal(t, [sha256.Size]byte{}, w.lastHash)

	w.check()
	assert.Equal(t, 1, called)
	assert.Equal(t, [sha256.Size]byte{}, w.lastHash, "hash must not be updated after handler error")

	w.check()
	assert.Equal(t, 2, called)
	assert.Equal(t, sha256.Sum256(content), w.lastHash, "hash must be updated after handler success")
}

func TestClampsInterval(t *testing.T) {
	handler := func(_ []byte) error { return nil }

	w := New("/nonexistent", time.Millisecond, handler)
	assert.GreaterOrEqual(t, w.interval, minInterval)

	w = New("/nonexistent", minInterval, handler)
	assert.Equal(t, minInterval, w.interval)

	longInterval := 2 * minInterval
	w = New("/nonexistent", longInterval, handler)
	assert.Equal(t, longInterval, w.interval)
}

func TestOversizedFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	oversizedContent := []byte(strings.Repeat("x", defaultMaxFileSize+1))
	require.NoError(t, os.WriteFile(filePath, oversizedContent, 0600))

	var called int
	handler := func(_ []byte) error { called++; return nil }

	var errorCount int
	w := New(filePath, 24*time.Hour, handler, WithOnError(func(_ error) { errorCount++ }))
	w.check()

	assert.Equal(t, 0, called, "handler must not be called for oversized files")
	assert.Equal(t, 1, errorCount, "onError must be called")
	assert.NotEqual(t, [sha256.Size]byte{}, w.lastHash)

	w.check()
	assert.Equal(t, 1, errorCount, "onError must not be called again for the same oversized file")
}

func TestCustomMaxFileSize(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	require.NoError(t, os.WriteFile(filePath, []byte("small"), 0600))

	var called int
	handler := func(_ []byte) error { called++; return nil }

	w := New(filePath, 24*time.Hour, handler, WithMaxFileSize(3))
	w.check()
	assert.Equal(t, 0, called, "handler must not be called when file exceeds custom max size")
}

func TestOnErrorCallback(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	require.NoError(t, os.WriteFile(filePath, []byte(strings.Repeat("x", defaultMaxFileSize+1)), 0600))

	var received []error
	w := New(filePath, 24*time.Hour, func(_ []byte) error { return nil },
		WithOnError(func(err error) { received = append(received, err) }))

	w.check()
	require.Len(t, received, 1)
	assert.Contains(t, received[0].Error(), "exceeds maximum size")
}

func TestStartStop(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0600))

	var called int
	w := New(filePath, 50*time.Millisecond, func(_ []byte) error { called++; return nil })
	w.Start()

	require.Eventually(t, func() bool { return called >= 1 }, 2*time.Second, 50*time.Millisecond,
		"handler was not called")

	done := make(chan struct{})
	go func() { w.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop within timeout")
	}
}

func TestHandlerReceivesFileContent(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	expected := `{"keys": [{"name": "test"}]}`
	require.NoError(t, os.WriteFile(filePath, []byte(expected), 0600))

	var received string
	w := New(filePath, 24*time.Hour, func(data []byte) error {
		received = string(data)
		return nil
	})
	w.check()
	assert.Equal(t, expected, received)
}

func TestNoOnErrorCallbackIsOK(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	require.NoError(t, os.WriteFile(filePath, []byte(strings.Repeat("x", defaultMaxFileSize+1)), 0600))

	w := New(filePath, 24*time.Hour, func(_ []byte) error { return nil })
	assert.NotPanics(t, func() { w.check() })
}

func TestOversizedFileChanges(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	content1 := strings.Repeat("a", defaultMaxFileSize+1)
	require.NoError(t, os.WriteFile(filePath, []byte(content1), 0600))

	var errorCount int
	w := New(filePath, 24*time.Hour, func(_ []byte) error { return nil },
		WithOnError(func(_ error) { errorCount++ }))

	w.check()
	assert.Equal(t, 1, errorCount)

	content2 := fmt.Sprintf("%s%s", strings.Repeat("b", defaultMaxFileSize+1), "extra")
	require.NoError(t, os.WriteFile(filePath, []byte(content2), 0600))

	w.check()
	assert.Equal(t, 2, errorCount, "onError must be called again when oversized file changes")
}
