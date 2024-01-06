package oomcheck

import (
	"io"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This value shows up when the memory limit isn't set. See the following article that describes it.
// https://unix.stackexchange.com/questions/420906/what-is-the-value-for-the-cgroups-limit-in-bytes-if-the-memory-is-not-restricte
const maxV1Limit = 9223372036854771712

func TestNew(t *testing.T) {
	r := NewMemoryUsageReader()

	rImpl, ok := r.(*memoryUsageReaderImpl)
	require.True(t, ok)
	assert.Equal(t, "/sys/fs/cgroup/memory/memory.stat", rImpl.v1StatFilePath)
	assert.Equal(t, "/sys/fs/cgroup/memory/memory.usage_in_bytes", rImpl.v1UsageFilePath)
	assert.Equal(t, "/proc/self/cgroup", rImpl.procCgroupFilePath)
	assert.ElementsMatch(t, []string{"/sys/fs/cgroup/unified", "/sys/fs/cgroup"}, rImpl.v2RootDirs)
}

func TestNoPanicWhenClosingWithoutOpen(t *testing.T) {
	r := NewMemoryUsageReader()
	// This essentially checks that closing nil file pointer does not lead to panic which is something provided by the
	// standard library implementation, however I don't want things to start dying in case the library implementation
	// changes.
	r.Close()
}

func TestGetUsageRealLike(t *testing.T) {
	cases := map[string]struct{ limit, used uint64 }{
		"cgroupv1-gke-node":          {limit: maxV1Limit, used: 1_445_191_680},
		"cgroupv1-gke-pod":           {limit: 2_147_483_648, used: 23_207_936},
		"cgroupv1-gke-pod-no-limits": {limit: 3_050_844_160, used: 25_706_496},
		"cgroupv1-minikube-node":     {limit: maxV1Limit, used: 5_519_925_248},
		"cgroupv1-minikube-pod":      {limit: 2_147_483_648, used: 61_210_624},
		"cgroupv1-ocp-node":          {limit: maxV1Limit, used: 13_440_061_440},
		"cgroupv1-ocp-pod":           {limit: 2_147_483_648, used: 22_138_880},
		"cgroupv2-crafted":           {limit: 6_291_456_000, used: 2_076_200_960},
		"cgroupv2-crafted-no-limits": {limit: 0xffffffffffffffff, used: 2_076_200_960},
	}

	for dir, expected := range cases {
		t.Run(dir, func(t *testing.T) {
			tmp := t.TempDir()

			source := path.Join("testfiles/real-fs", dir)
			setupTestDir(t, source, tmp)

			reader := newWithRoot(tmp)
			require.NoError(t, reader.Open())
			defer reader.Close()

			usage, err := reader.GetUsage()

			assert.NoError(t, err)
			assert.Equal(t, expected.used, usage.Used)
			assert.Equal(t, expected.limit, usage.Limit)
		})
	}
}

func setupTestDir(t require.TestingT, source, dest string) {
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

func copyFiles(t require.TestingT, srcDir, destDir string) {
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

func closeFile(t require.TestingT, file *os.File) {
	require.NoError(t, file.Close())
}

func TestReadChangingValueV1andV2(t *testing.T) {
	assert.Fail(t, "implement me")
}

func TestSynthetic(t *testing.T) {
	cases := map[string]struct {
		fileContents map[string]string
		openErrors   []string
		getErrors    []string
	}{
		"nothing exists": {
			openErrors: []string{"cgroupv1", "cgroupv2"},
		},
		"only memory.stat exists but empty": {
			fileContents: map[string]string{
				"/sys/fs/cgroup/memory/memory.stat": "",
			},
			getErrors: []string{"cgroupv1", "EOF"},
		},
		"only memory.stat with garbage1": {
			fileContents: map[string]string{
				"/sys/fs/cgroup/memory/memory.stat": "  ",
			},
			getErrors: []string{"reading cgroupv1", "invalid syntax"},
		},
		"only memory.stat with garbage": {
			fileContents: map[string]string{
				"/sys/fs/cgroup/memory/memory.stat": randString(t, 1000),
			},
			getErrors: []string{"reading cgroupv1", "invalid syntax"},
		},
		"huge memory.stat with garbage": {
			fileContents: map[string]string{
				"/sys/fs/cgroup/memory/memory.stat": randString(t, 987_654),
			},
			getErrors: []string{"reading cgroupv1", "invalid syntax"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			writeFiles(t, tmp, c.fileContents)

			reader := newWithRoot(tmp)
			err := reader.Open()
			assertExpectedError(t, err, c.openErrors)
			if len(c.openErrors) != 0 {
				return
			}
			defer reader.Close()

			usage, err := reader.GetUsage()
			assertExpectedError(t, err, c.getErrors)
			if len(c.getErrors) != 0 {
				return
			}

			assert.Equal(t, uint64(0), usage.Used)
			assert.Equal(t, uint64(0), usage.Limit)
		})
	}
}

func writeFiles(t *testing.T, dir string, fileContents map[string]string) {
	for file, contents := range fileContents {
		fullPath := path.Join(dir, file)
		require.NoError(t, os.MkdirAll(path.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, ([]byte)(contents), 0644))
	}
}

func assertExpectedError(t *testing.T, actualErr error, expectedContents []string) {
	if len(expectedContents) == 0 {
		assert.NoError(t, actualErr)
	}
	for _, fragment := range expectedContents {
		assert.ErrorContains(t, actualErr, fragment)
	}
}

func randString(t *testing.T, size int) string {
	buf := make([]byte, size)
	n, err := rand.Read(buf[:])
	require.NoError(t, err)
	assert.Equal(t, size, n)
	return string(buf[:n])
}

func BenchmarkReadV1(b *testing.B) {
	tmp := b.TempDir()
	setupTestDir(b, "testfiles/real-fs/cgroupv1-ocp-pod", tmp)

	reader := newWithRoot(tmp)
	require.NoError(b, reader.Open())
	defer reader.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage, err := reader.GetUsage()

		assert.NoError(b, err)
		assert.Equal(b, uint64(22_138_880), usage.Used)
		assert.Equal(b, uint64(2_147_483_648), usage.Limit)
	}
}

func BenchmarkReadV2(b *testing.B) {
	tmp := b.TempDir()
	setupTestDir(b, "testfiles/real-fs/cgroupv2-crafted", tmp)

	reader := newWithRoot(tmp)
	require.NoError(b, reader.Open())
	defer reader.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage, err := reader.GetUsage()

		assert.NoError(b, err)
		assert.Equal(b, uint64(2_076_200_960), usage.Used)
		assert.Equal(b, uint64(6_291_456_000), usage.Limit)
	}
}

func BenchmarkReopen(b *testing.B) {
	const base uint64 = 0

	for i := 0; i < b.N; i++ {
		data, err := os.ReadFile("/sys/fs/cgroup/system.slice/snapd.service/memory.current")
		require.NoError(b, err)

		val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		require.NoError(b, err)

		assert.Greater(b, val, base)
	}
}

func BenchmarkReread(b *testing.B) {
	const base uint64 = 0

	fd, err := os.OpenFile("/sys/fs/cgroup/system.slice/snapd.service/memory.current", os.O_RDONLY, 0)
	defer fd.Close()
	require.NoError(b, err)
	b.ResetTimer()

	data := make([]byte, 1024)
	for i := 0; i < b.N; i++ {

		_, err = fd.Seek(0, 0)
		require.NoError(b, err)

		n, err := fd.Read(data)
		require.NoError(b, err)
		require.Greater(b, n, 0)

		val, err := strconv.ParseUint(strings.TrimSpace(string(data[:n])), 10, 64)
		require.NoError(b, err)

		assert.Greater(b, val, base)
	}
}
