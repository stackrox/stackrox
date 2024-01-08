package metrics

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCGroup1(t *testing.T) {
	fsys := os.DirFS("testdata/throttle/cgroup1")
	_, err := fs.Stat(fsys, cgroup2CPUStatFile)
	require.True(t, errors.Is(err, fs.ErrNotExist))
	b, err := fs.ReadFile(fsys, cgroupCPUStatFile)
	require.NoError(t, err)

	vals := parseStats(cgroupCPUStatFile, b, cgroupStats)
	assert.Equal(t, 10, vals.periods)
	assert.Equal(t, 11, vals.throttled)
	assert.Equal(t, 1208573224, vals.throttledTime)
}

func TestCGroup2(t *testing.T) {
	fsys := os.DirFS("testdata/throttle/cgroup2")
	_, err := fs.Stat(fsys, cgroupCPUStatFile)
	require.True(t, errors.Is(err, fs.ErrNotExist))
	b, err := fs.ReadFile(fsys, cgroup2CPUStatFile)
	require.NoError(t, err)

	vals := parseStats(cgroup2CPUStatFile, b, cgroup2Stats)
	assert.Equal(t, 39, vals.periods)
	assert.Equal(t, 20, vals.throttled)
	assert.Equal(t, 22, vals.throttledTime)
}
