package hostprobe

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestHasAnyRepoFile(t *testing.T) {
	t.Run("finds repo file in single dir", func(t *testing.T) {
		fsys := fstest.MapFS{
			"etc/yum.repos.d/baseos.repo": &fstest.MapFile{Data: []byte("[baseos]")},
			"etc/yum.repos.d/README.txt":  &fstest.MapFile{Data: []byte("not a repo")},
		}
		found, err := HasAnyRepoFile(fsys, []string{YumReposDirPath})
		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("finds repo file across multiple dirs", func(t *testing.T) {
		fsys := fstest.MapFS{
			"usr/share/dnf5/repos.d/fedora.repo": &fstest.MapFile{Data: []byte("[fedora]")},
		}
		found, err := HasAnyRepoFile(fsys, []string{YumReposDirPath, DNF5ReposDirPath})
		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("returns false when dirs exist but have no repo files", func(t *testing.T) {
		fsys := fstest.MapFS{
			"etc/yum.repos.d/README.txt": &fstest.MapFile{Data: []byte("not a repo")},
		}
		found, err := HasAnyRepoFile(fsys, []string{YumReposDirPath})
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("returns false with error when all dirs missing", func(t *testing.T) {
		fsys := fstest.MapFS{}
		found, err := HasAnyRepoFile(fsys, []string{YumReposDirPath, DNF5ReposDirPath})
		require.Error(t, err)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.False(t, found)
	})

	t.Run("skips unreadable dir and finds repo in another", func(t *testing.T) {
		fsys := &unreadableDirFS{
			FS:  fstest.MapFS{"usr/share/dnf5/repos.d/fedora.repo": &fstest.MapFile{Data: []byte("[fedora]")}},
			dir: "etc/yum.repos.d",
			err: fs.ErrPermission,
		}
		found, err := HasAnyRepoFile(fsys, []string{YumReposDirPath, DNF5ReposDirPath})
		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("returns false when all dirs fail", func(t *testing.T) {
		fsys := &unreadableDirFS{
			FS:  fstest.MapFS{},
			dir: "etc/yum.repos.d",
			err: fs.ErrPermission,
		}
		found, err := HasAnyRepoFile(fsys, []string{YumReposDirPath})
		require.Error(t, err)
		require.ErrorIs(t, err, fs.ErrPermission)
		require.False(t, found)
	})

	t.Run("returns partial ReadDir errors when some dirs fail and none have repos", func(t *testing.T) {
		fsys := &unreadableDirFS{
			FS: fstest.MapFS{
				"etc/yum.repos.d/README.txt": &fstest.MapFile{Data: []byte("x")},
			},
			dir: "usr/share/dnf5/repos.d",
			err: fs.ErrPermission,
		}
		found, err := HasAnyRepoFile(fsys, []string{YumReposDirPath, DNF5ReposDirPath})
		require.Error(t, err)
		require.ErrorIs(t, err, fs.ErrPermission)
		require.False(t, found)
	})
}

// unreadableDirFS wraps an fs.FS to return a fixed error for ReadDir on a specific path
type unreadableDirFS struct {
	fs.FS
	// return `err` when reading from `dir`
	dir string
	err error
}

func (f *unreadableDirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == f.dir {
		return nil, f.err
	}
	return fs.ReadDir(f.FS, name)
}

func TestHostPathFor(t *testing.T) {
	t.Run("empty host path uses original path", func(t *testing.T) {
		require.Equal(t, "/etc/os-release", HostPathFor("", "/etc/os-release"))
	})

	t.Run("root host path uses original path", func(t *testing.T) {
		require.Equal(t, "/etc/os-release", HostPathFor("/", "/etc/os-release"))
	})

	t.Run("prefixes host path and cleans", func(t *testing.T) {
		require.Equal(t, "/host/etc/os-release", HostPathFor("/host//", "/etc/os-release"))
		require.Equal(t, "/host/var/cache/dnf", HostPathFor("/root/../host", "/var/lib/../cache//dnf/"))
	})
}

func TestFileExists(t *testing.T) {
	hostPath := t.TempDir()
	target := HostPathFor(hostPath, "/etc/os-release")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("ID=rhel"), 0o644))

	require.FileExists(t, HostPathFor(hostPath, "/etc/os-release"))
	require.NoFileExists(t, HostPathFor(hostPath, "/etc/missing"))
}

func TestDetectDNFVersion(t *testing.T) {
	t.Run("unknown when no history db exists", func(t *testing.T) {
		hostPath := t.TempDir()
		require.Equal(t, DNFVersionUnknown, DetectDNFVersion(hostPath))
	})

	t.Run("detects dnf4", func(t *testing.T) {
		hostPath := t.TempDir()
		p := HostPathFor(hostPath, DNF4HistoryDBPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte("db"), 0o644))
		require.Equal(t, DNFVersion4, DetectDNFVersion(hostPath))
	})

	t.Run("detects dnf5", func(t *testing.T) {
		hostPath := t.TempDir()
		p := HostPathFor(hostPath, DNF5HistoryDBPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte("db"), 0o644))
		require.Equal(t, DNFVersion5, DetectDNFVersion(hostPath))
	})

	t.Run("prefers dnf5 when both exist", func(t *testing.T) {
		hostPath := t.TempDir()
		p4 := HostPathFor(hostPath, DNF4HistoryDBPath)
		p5 := HostPathFor(hostPath, DNF5HistoryDBPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(p4), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Dir(p5), 0o755))
		require.NoError(t, os.WriteFile(p4, []byte("db4"), 0o644))
		require.NoError(t, os.WriteFile(p5, []byte("db5"), 0o644))
		require.Equal(t, DNFVersion5, DetectDNFVersion(hostPath))
	})

	t.Run("falls back to dnf4 when dnf5 path is broken symlink", func(t *testing.T) {
		hostPath := t.TempDir()
		p4 := HostPathFor(hostPath, DNF4HistoryDBPath)
		p5 := HostPathFor(hostPath, DNF5HistoryDBPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(p4), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Dir(p5), 0o755))
		require.NoError(t, os.WriteFile(p4, []byte("db4"), 0o644))
		require.NoError(t, os.Symlink(filepath.Join(hostPath, "does-not-exist.sqlite"), p5))

		require.Equal(t, DNFVersion4, DetectDNFVersion(hostPath))
	})
}

func Test_hasEntitlementCertKeyPairAtPath(t *testing.T) {
	t.Run("returns true when matching pair exists", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "111-key.pem"), []byte("k"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "111.pem"), []byte("c"), 0o644))
		ok, err := hasEntitlementCertKeyPairAtPath(dir)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("returns false for mismatched files", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "111-key.pem"), []byte("k"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "222.pem"), []byte("c"), 0o644))
		ok, err := hasEntitlementCertKeyPairAtPath(dir)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("ignores directories and returns true when valid pair exists", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "111-key.pem"), []byte("k"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "111.pem"), []byte("c"), 0o644))
		ok, err := hasEntitlementCertKeyPairAtPath(dir)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("returns error for missing directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing")
		ok, err := hasEntitlementCertKeyPairAtPath(dir)
		require.Error(t, err)
		require.False(t, ok)
	})

	t.Run("returns error when path is file, not directory", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "entitlement")
		require.NoError(t, os.WriteFile(filePath, []byte("not a dir"), 0o644))

		ok, err := hasEntitlementCertKeyPairAtPath(filePath)
		require.Error(t, err)
		require.False(t, ok)
	})
}

func TestHasEntitlementCertKeyPair(t *testing.T) {
	hostPath := t.TempDir()
	entitlementDir := HostPathFor(hostPath, EntitlementDirPath)
	require.NoError(t, os.MkdirAll(entitlementDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(entitlementDir, "3341241341658386286-key.pem"), []byte("k"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(entitlementDir, "3341241341658386286.pem"), []byte("c"), 0o644))

	ok, err := HasEntitlementCertKeyPair(hostPath)
	require.NoError(t, err)
	require.True(t, ok)
}
