package oomcheck

import (
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This value shows up when the memory limit isn't set. See the following article that describes it.
// https://unix.stackexchange.com/questions/420906/what-is-the-value-for-the-cgroups-limit-in-bytes-if-the-memory-is-not-restricte
const maxLimit = 9223372036854771712

func TestNew(t *testing.T) {
	r := NewMemoryUsageReader()

	rImpl, ok := r.(*memoryUsageReaderImpl)
	require.True(t, ok)
	assert.Equal(t, "/sys/fs/cgroup/memory/memory.stat", rImpl.v1StatFile)
	assert.Equal(t, "/sys/fs/cgroup/memory/memory.usage_in_bytes", rImpl.v1UsageFile)
	assert.Equal(t, "/proc/self/cgroup", rImpl.procCgroupFile)
	assert.ElementsMatch(t, []string{"/sys/fs/cgroup/unified", "/sys/fs/cgroup"}, rImpl.v2RootDirs)
}

func TestGetUsageCgroupV1(t *testing.T) {
	cases := map[string]struct{ limit, used uint64 }{
		"cgroupv1-gke-node":          {limit: maxLimit, used: 1_445_191_680},
		"cgroupv1-gke-pod":           {limit: 2_147_483_648, used: 23_207_936},
		"cgroupv1-gke-pod-no-limits": {limit: 3_050_844_160, used: 25_706_496},
		"cgroupv1-minikube-node":     {limit: maxLimit, used: 5_519_925_248},
		"cgroupv1-minikube-pod":      {limit: 2_147_483_648, used: 61_210_624},
		"cgroupv1-ocp-node":          {limit: maxLimit, used: 13_440_061_440},
		"cgroupv1-ocp-pod":           {limit: 2_147_483_648, used: 22_138_880},
	}

	for dir, expected := range cases {
		t.Run(dir, func(t *testing.T) {
			tmp := t.TempDir()

			source := path.Join("testfiles/real-fs", dir)
			setupTestDir(t, source, tmp)

			reader := newWithRoot(tmp)

			usage, err := reader.GetUsage()

			assert.NoError(t, err)
			assert.Equal(t, expected.used, usage.Used)
			assert.Equal(t, expected.limit, usage.Limit)
		})
	}
}

func TestGetUsageCgroupV2(t *testing.T) {
	tmp := t.TempDir()

	source := path.Join("testfiles/real-fs", "cgroupv2-crafted")
	setupTestDir(t, source, tmp)

	reader := newWithRoot(tmp)
	usage, err := reader.GetUsage()

	assert.NoError(t, err)
	assert.Equal(t, 2_076_200_960, usage.Used)
	assert.Equal(t, 6_291_456_000, usage.Limit)
}

func setupTestDir(t *testing.T, source, dest string) {
	entries, err := os.ReadDir(source)
	require.NoError(t, err)
	for _, subdir := range entries {
		if !subdir.IsDir() {
			continue
		}

		srcDir := path.Join(source, subdir.Name())
		// Underscores have special meaning: we convert them to path separators.
		// Why? In order to have more flat directory hierarchy in tests.
		destDir := path.Join(dest, path.Join(strings.Split(subdir.Name(), "_")...))

		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		copyFiles(t, srcDir, destDir)
	}
}

func copyFiles(t *testing.T, srcDir, destDir string) {
	entries, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	for _, file := range entries {
		if !file.Type().IsRegular() {
			continue
		}

		srcPath := path.Join(srcDir, file.Name())
		dstPath := path.Join(destDir, file.Name())

		in, err := os.OpenFile(srcPath, os.O_RDONLY, 0644)
		require.NoError(t, err)
		defer closeFile(t, in)

		out, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		require.NoError(t, err)
		defer closeFile(t, out)

		_, err = io.Copy(out, in)
		require.NoError(t, err)
	}
}

func closeFile(t *testing.T, file *os.File) {
	require.NoError(t, file.Close())
}
