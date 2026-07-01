package discovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverDnfRepoFilePresence(t *testing.T) {
	tests := map[string]struct {
		setup            func(t *testing.T) (string, []string)
		expectedHasDir   bool
		expectedHasRepo  bool
		expectedErrParts []string
	}{
		"should return true when repo file exists in reposdir": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				reposDirPath := "/etc/yum.repos.d"
				reposPath := hostprobe.HostPathFor(hostPath, reposDirPath)
				require.NoError(t, os.MkdirAll(reposPath, 0750))
				repoFilePath := filepath.Join(reposPath, "test.repo")
				require.NoError(t, os.WriteFile(repoFilePath, []byte("content"), 0600))
				return hostPath, []string{reposDirPath}
			},
			expectedHasDir:  true,
			expectedHasRepo: true,
		},
		"should return error when all reposdirs are unreadable": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				return hostPath, []string{"/etc/yum.repos.d", "/etc/yum/repos.d"}
			},
			expectedHasDir:   false,
			expectedHasRepo:  false,
			expectedErrParts: []string{"reading", "no such file or directory"},
		},
		"should return error when reposdirs are readable but no repo files exist": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				reposDirPaths := []string{"/etc/yum.repos.d", "/etc/yum/repos.d"}
				for _, reposDirPath := range reposDirPaths {
					reposPath := hostprobe.HostPathFor(hostPath, reposDirPath)
					require.NoError(t, os.MkdirAll(reposPath, 0750))
					require.NoError(t, os.WriteFile(filepath.Join(reposPath, "not-a-repo.txt"), []byte("content"), 0600))
				}
				return hostPath, reposDirPaths
			},
			expectedHasDir:   true,
			expectedHasRepo:  false,
			expectedErrParts: []string{"no .repo files found"},
		},
		"should return true when repo file exists even if other reposdir is missing": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				reposDirPaths := []string{"/etc/yum.repos.d", "/etc/yum/repos.d"}
				reposPath := hostprobe.HostPathFor(hostPath, reposDirPaths[0])
				require.NoError(t, os.MkdirAll(reposPath, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(reposPath, "example.repo"), []byte("content"), 0600))
				return hostPath, reposDirPaths
			},
			expectedHasDir:  true,
			expectedHasRepo: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hostPath, reposDirPaths := tt.setup(t)
			hasDir, hasRepo, err := discoverDnfRepoFilePresence(hostPath, reposDirPaths)
			assert.Equal(t, tt.expectedHasDir, hasDir)
			assert.Equal(t, tt.expectedHasRepo, hasRepo)
			if len(tt.expectedErrParts) == 0 {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, part := range tt.expectedErrParts {
				assert.Contains(t, err.Error(), part)
			}
		})
	}
}

func TestDiscoverDnf4CacheRepoDirPresence(t *testing.T) {
	tests := map[string]struct {
		setup            func(t *testing.T) (hostPath string, cacheDirPath string)
		expectedHasDir   bool
		expectedFound    bool
		expectedErrParts []string
	}{
		"true when cache dir contains repo directory": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				cacheDirPath := "/var/cache/dnf"
				cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
				require.NoError(t, os.MkdirAll(filepath.Join(cachePath, "rhel-9-for-x86_64-appstream-rpms-123"), 0750))
				return hostPath, cacheDirPath
			},
			expectedHasDir: true,
			expectedFound:  true,
		},
		"false when cache dir has no repo directories": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				cacheDirPath := "/var/cache/dnf"
				cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
				require.NoError(t, os.MkdirAll(cachePath, 0750))
				return hostPath, cacheDirPath
			},
			expectedHasDir: true,
			expectedFound:  false,
		},
		"false when cache has subdirs but none match -rpms- pattern": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				cacheDirPath := "/var/cache/dnf"
				cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
				require.NoError(t, os.MkdirAll(filepath.Join(cachePath, "fedora"), 0750))
				return hostPath, cacheDirPath
			},
			expectedHasDir: true,
			expectedFound:  false,
		},
		"error when cache dir is missing": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				return hostPath, "/var/cache/dnf"
			},
			expectedHasDir:   false,
			expectedFound:    false,
			expectedErrParts: []string{"reading"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hostPath, cacheDirPath := tt.setup(t)
			hasDir, found, err := discoverDnf4CacheRepoDirPresence(hostPath, cacheDirPath)
			assert.Equal(t, tt.expectedHasDir, hasDir)
			assert.Equal(t, tt.expectedFound, found)
			if len(tt.expectedErrParts) == 0 {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, part := range tt.expectedErrParts {
				assert.Contains(t, err.Error(), part)
			}
		})
	}
}

func TestDiscoverDnf5CacheRepoDirPresence(t *testing.T) {
	tests := map[string]struct {
		setup            func(t *testing.T) (hostPath string, cacheDirPath string)
		expectedHasDir   bool
		expectedFound    bool
		expectedErrParts []string
	}{
		"true when cache dir has repository entries": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				cacheDirPath := "/var/cache/libdnf5"
				cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
				require.NoError(t, os.MkdirAll(filepath.Join(cachePath, "fedora"), 0750))
				return hostPath, cacheDirPath
			},
			expectedHasDir: true,
			expectedFound:  true,
		},
		"false when cache root exists but has no subdirectories": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				cacheDirPath := "/var/cache/libdnf5"
				cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
				require.NoError(t, os.MkdirAll(cachePath, 0750))
				return hostPath, cacheDirPath
			},
			expectedHasDir: true,
			expectedFound:  false,
		},
		"error when cache dir is missing": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				return hostPath, "/var/cache/libdnf5"
			},
			expectedHasDir:   false,
			expectedFound:    false,
			expectedErrParts: []string{"reading"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hostPath, cacheDirPath := tt.setup(t)
			hasDir, found, err := discoverDnf5CacheRepoDirPresence(hostPath, cacheDirPath)
			assert.Equal(t, tt.expectedHasDir, hasDir)
			assert.Equal(t, tt.expectedFound, found)
			if len(tt.expectedErrParts) == 0 {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, part := range tt.expectedErrParts {
				assert.Contains(t, err.Error(), part)
			}
		})
	}
}

// TestDiscoverDnfStatusFlags exercises discoverDnfStatusFlags's flag-combination
// and error-aggregation logic against synthetic directory layouts identified by
// arbitrary absolute paths. History-DB detection (which depends on hostPath,
// not on the repos/cache args) is covered separately by
// TestDiscoverDnfStatusFlags_RealPaths.
func TestDiscoverDnfStatusFlags(t *testing.T) {
	tests := map[string]struct {
		setup         func(t *testing.T) (dnf4Repos, dnf5Repos []string, dnf4Cache, dnf5Cache string)
		expectedFlags []v1.DnfStatusFlag
		errorContains string
	}{
		"repo and v4 cache found": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				repoDir := filepath.Join(tmp, "yum.repos.d")
				require.NoError(t, os.MkdirAll(repoDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(repoDir, "rhel9.repo"), []byte("[repo]"), 0644))
				cacheDir := filepath.Join(tmp, "dnf")
				require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, "rhel-9-for-x86_64-appstream-rpms-abc"), 0755))
				return []string{repoDir}, nil, cacheDir, ""
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
		},
		"repo and v5 cache found": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				repoDir := filepath.Join(tmp, "repos.d")
				require.NoError(t, os.MkdirAll(repoDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(repoDir, "fedora.repo"), []byte("[repo]"), 0644))
				cacheDir := filepath.Join(tmp, "libdnf5")
				require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, "fedora"), 0755))
				return nil, []string{repoDir}, "", cacheDir
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V5_CACHE_FOUND,
			},
		},
		"repo found in second of two reposdirs": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				emptyDir := filepath.Join(tmp, "yum.repos.d")
				repoDir := filepath.Join(tmp, "yum", "repos.d")
				require.NoError(t, os.MkdirAll(emptyDir, 0755))
				require.NoError(t, os.MkdirAll(repoDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(repoDir, "rhel9.repo"), []byte("[repo]"), 0644))
				return []string{emptyDir, repoDir}, nil, "", ""
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
			},
		},
		"cache dir readable but no -rpms- subdirs: no cache flag, no error": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				repoDir := filepath.Join(tmp, "yum.repos.d")
				require.NoError(t, os.MkdirAll(repoDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(repoDir, "rhel9.repo"), []byte("[repo]"), 0644))
				cacheDir := filepath.Join(tmp, "dnf")
				require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, "fedora"), 0755))
				return []string{repoDir}, nil, cacheDir, ""
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
			},
		},
		"no repo files, cache exists": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				repoDir := filepath.Join(tmp, "yum.repos.d")
				require.NoError(t, os.MkdirAll(repoDir, 0755))
				cacheDir := filepath.Join(tmp, "dnf")
				require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, "rhel-9-for-x86_64-appstream-rpms-abc"), 0755))
				return []string{repoDir}, nil, cacheDir, ""
			},
			expectedFlags: []v1.DnfStatusFlag{v1.DnfStatusFlag_DNF_V4_CACHE_FOUND},
			errorContains: "no .repo files found",
		},
		"reposdir does not exist, cache exists": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				cacheDir := filepath.Join(tmp, "dnf")
				require.NoError(t, os.MkdirAll(cacheDir, 0755))
				return []string{filepath.Join(tmp, "nonexistent-repos")}, nil, cacheDir, ""
			},
			expectedFlags: nil,
			errorContains: "reading",
		},
		"repos exist, cache dir does not exist": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				repoDir := filepath.Join(tmp, "yum.repos.d")
				require.NoError(t, os.MkdirAll(repoDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(repoDir, "rhel9.repo"), []byte("[repo]"), 0644))
				return []string{repoDir}, nil, filepath.Join(tmp, "nonexistent-cache"), ""
			},
			expectedFlags: []v1.DnfStatusFlag{v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND},
			errorContains: "reading",
		},
		"neither reposdir nor cache dir exist": {
			setup: func(t *testing.T) ([]string, []string, string, string) {
				tmp := t.TempDir()
				return []string{filepath.Join(tmp, "yum.repos.d")}, nil, filepath.Join(tmp, "dnf"), ""
			},
			expectedFlags: nil,
			errorContains: "reading",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dnf4Repos, dnf5Repos, dnf4Cache, dnf5Cache := tt.setup(t)
			flags, err := discoverDnfStatusFlags("", dnf4Repos, dnf5Repos, dnf4Cache, dnf5Cache)
			assert.Equal(t, tt.expectedFlags, flags)
			if tt.errorContains == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.errorContains)
		})
	}
}

// TestDiscoverDnfStatusFlags_RealPaths is an end-to-end smoke test that runs
// discoverDnfStatusFlags against the actual hostprobe path constants used in
// production, including history-DB detection (which is keyed off hostPath).
func TestDiscoverDnfStatusFlags_RealPaths(t *testing.T) {
	tests := map[string]struct {
		setup         func(t *testing.T) string
		expectedFlags []v1.DnfStatusFlag
		expectedErr   error
	}{
		"all flags present (dnf4)": {
			setup: func(t *testing.T) string {
				hp := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "etc/yum.repos.d"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(hp, "etc/yum.repos.d/redhat.repo"), []byte("[baseos]"), 0644))
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "var/cache/dnf/rhel-9-for-x86_64-baseos-rpms-abc"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "var/lib/dnf"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(hp, "var/lib/dnf/history.sqlite"), []byte("db"), 0644))
				return hp
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
				v1.DnfStatusFlag_DNF_V4_HISTORY_DB_FOUND,
			},
		},
		"all flags present (dnf5)": {
			setup: func(t *testing.T) string {
				hp := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "usr/share/dnf5/repos.d"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(hp, "usr/share/dnf5/repos.d/redhat.repo"), []byte("[baseos]"), 0644))
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "var/cache/libdnf5/fedora"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "usr/lib/sysimage/libdnf5"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(hp, "usr/lib/sysimage/libdnf5/transaction_history.sqlite"), []byte("db"), 0644))
				return hp
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V5_CACHE_FOUND,
				v1.DnfStatusFlag_DNF_V5_HISTORY_DB_FOUND,
			},
		},
		"repo config and cache present but no history DB (cloud image)": {
			setup: func(t *testing.T) string {
				hp := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "etc/yum.repos.d"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(hp, "etc/yum.repos.d/redhat.repo"), []byte("[baseos]"), 0644))
				require.NoError(t, os.MkdirAll(filepath.Join(hp, "var/cache/dnf/rhel-9-for-x86_64-baseos-rpms-abc"), 0755))
				return hp
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
		},
		"empty host path": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedFlags: nil,
			expectedErr:   fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hostPath := tt.setup(t)
			flags, err := discoverDnfStatusFlags(hostPath,
				hostprobe.DNF4ReposDirs, []string{hostprobe.DNF5ReposDirPath},
				hostprobe.DNF4CacheDirPath, hostprobe.DNF5CacheDirPath,
			)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedFlags, flags)
		})
	}
}

func TestDeriveLegacyDnfMetadataStatus(t *testing.T) {
	tests := map[string]struct {
		flags    []v1.DnfStatusFlag
		expected v1.DnfMetadataStatus
	}{
		"nil flags": {
			flags:    nil,
			expected: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
		},
		"empty flags": {
			flags:    []v1.DnfStatusFlag{},
			expected: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
		},
		"repo and v4 cache": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
			expected: v1.DnfMetadataStatus_AVAILABLE,
		},
		"repo and v5 cache": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V5_CACHE_FOUND,
			},
			expected: v1.DnfMetadataStatus_AVAILABLE,
		},
		"repo and cache with history flags": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
				v1.DnfStatusFlag_DNF_V4_HISTORY_DB_FOUND,
			},
			expected: v1.DnfMetadataStatus_AVAILABLE,
		},
		"repo only": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
			},
			expected: v1.DnfMetadataStatus_UNAVAILABLE,
		},
		"v4 cache only": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
			expected: v1.DnfMetadataStatus_UNAVAILABLE,
		},
		"v5 cache only": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_V5_CACHE_FOUND,
			},
			expected: v1.DnfMetadataStatus_UNAVAILABLE,
		},
		"both caches, no repo": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
				v1.DnfStatusFlag_DNF_V5_CACHE_FOUND,
			},
			expected: v1.DnfMetadataStatus_UNAVAILABLE,
		},
		"history db only": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_V5_HISTORY_DB_FOUND,
			},
			expected: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, deriveLegacyDnfMetadataStatus(tt.flags))
		})
	}
}
