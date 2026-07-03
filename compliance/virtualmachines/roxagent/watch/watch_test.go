package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const eventWaitTimeout = 2 * time.Second

func TestNew_NoCandidateDirectory_ReturnsError(t *testing.T) {
	hostPath := t.TempDir() // empty: neither candidate directory exists

	w, err := New(hostPath)
	require.Error(t, err)
	require.Nil(t, w)
}

func TestNew_WatchesLegacyRPMDir(t *testing.T) {
	hostPath := t.TempDir()
	rpmDir := hostprobe.HostPathFor(hostPath, "/var/lib/rpm")
	require.NoError(t, os.MkdirAll(rpmDir, 0o755))

	w, err := New(hostPath)
	require.NoError(t, err)
	require.NotNil(t, w)
	defer func() { _ = w.Close() }()

	require.NoError(t, os.WriteFile(filepath.Join(rpmDir, "rpmdb.sqlite"), []byte("x"), 0o644))

	select {
	case <-w.Triggered():
	case <-time.After(eventWaitTimeout):
		t.Fatal("expected a trigger after writing to the watched directory")
	}
}

func TestNew_PrefersModernRPMDirWhenBothExist(t *testing.T) {
	hostPath := t.TempDir()
	modernDir := hostprobe.HostPathFor(hostPath, "/usr/lib/sysimage/rpm")
	legacyDir := hostprobe.HostPathFor(hostPath, "/var/lib/rpm")
	require.NoError(t, os.MkdirAll(modernDir, 0o755))
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))

	w, err := New(hostPath)
	require.NoError(t, err)
	require.NotNil(t, w)
	defer func() { _ = w.Close() }()

	// A write in the non-preferred legacy dir must NOT be seen, proving the
	// modern (first candidate) directory is the one actually being watched.
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "ignored"), []byte("x"), 0o644))
	select {
	case <-w.Triggered():
		t.Fatal("should not see events from the non-preferred legacy directory")
	case <-time.After(200 * time.Millisecond):
	}

	require.NoError(t, os.WriteFile(filepath.Join(modernDir, "rpmdb.sqlite"), []byte("x"), 0o644))
	select {
	case <-w.Triggered():
	case <-time.After(eventWaitTimeout):
		t.Fatal("expected a trigger after writing to the preferred modern directory")
	}
}

func TestWatcher_CoalescesBurstOfEvents(t *testing.T) {
	hostPath := t.TempDir()
	rpmDir := hostprobe.HostPathFor(hostPath, "/var/lib/rpm")
	require.NoError(t, os.MkdirAll(rpmDir, 0o755))

	w, err := New(hostPath)
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	// Simulate a whole RPM transaction's worth of rapid writes.
	for i := range 10 {
		require.NoError(t, os.WriteFile(filepath.Join(rpmDir, fmt.Sprintf("f%d", i)), []byte("x"), 0o644))
	}

	select {
	case <-w.Triggered():
	case <-time.After(eventWaitTimeout):
		t.Fatal("expected a trigger after the write burst")
	}

	// The burst must collapse into exactly one pending trigger: no second
	// value should already be queued behind it.
	select {
	case <-w.Triggered():
		t.Fatal("burst of events should collapse into a single trigger, not a backlog")
	default:
	}
}

func TestIsRelevant(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		op       fsnotify.Op
		expected bool
	}{
		"write is relevant":      {op: fsnotify.Write, expected: true},
		"create is relevant":     {op: fsnotify.Create, expected: true},
		"rename is relevant":     {op: fsnotify.Rename, expected: true},
		"chmod is not relevant":  {op: fsnotify.Chmod, expected: false},
		"remove is not relevant": {op: fsnotify.Remove, expected: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isRelevant(fsnotify.Event{Op: tc.op}))
		})
	}
}
