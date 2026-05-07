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

func TestDiscoverDnfStatusFlags_Integration(t *testing.T) {
	tests := map[string]struct {
		dnf4ReposDirs []string
		dnf5ReposDirs []string
		repoDirFiles  map[string][]string
		cacheRootDir  string
		cacheDirs     []string
		isDNF5Cache   bool
		expectedFlags []v1.DnfStatusFlag
		expectedErrs  []string
	}{
		"repo and v4 cache found": {
			dnf4ReposDirs: []string{"yum.repos.d"},
			repoDirFiles:  map[string][]string{"yum.repos.d": {"rhel9.repo"}},
			cacheRootDir:  "dnf",
			cacheDirs:     []string{"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476"},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
		},
		"repo and v5 cache found": {
			dnf5ReposDirs: []string{"usr/share/dnf5/repos.d"},
			repoDirFiles:  map[string][]string{"usr/share/dnf5/repos.d": {"fedora.repo"}},
			cacheRootDir:  "var/cache/libdnf5",
			cacheDirs:     []string{"fedora"},
			isDNF5Cache:   true,
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V5_CACHE_FOUND,
			},
		},
		"repo found in second reposdir": {
			dnf4ReposDirs: []string{"yum.repos.d", "yum/repos.d"},
			repoDirFiles: map[string][]string{
				"yum.repos.d": {},
				"yum/repos.d": {"rhel9.repo"},
			},
			cacheRootDir: "dnf",
			cacheDirs:    []string{"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476"},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
		},
		"multiple repo files and cache dirs": {
			dnf4ReposDirs: []string{"yum.repos.d"},
			repoDirFiles: map[string][]string{
				"yum.repos.d": {"baseos.repo", "appstream.repo"},
			},
			cacheRootDir: "dnf",
			cacheDirs: []string{
				"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476",
				"rhel-9-for-x86_64-baseos-rpms-a2cdae14f4ed6c20",
			},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
		},
		"no repo files, cache exists": {
			dnf4ReposDirs: []string{"yum.repos.d"},
			repoDirFiles:  map[string][]string{"yum.repos.d": {}},
			cacheRootDir:  "dnf",
			cacheDirs:     []string{"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476"},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
			expectedErrs: []string{"no .repo files found"},
		},
		"repo found, empty cache": {
			dnf4ReposDirs: []string{"yum.repos.d"},
			repoDirFiles:  map[string][]string{"yum.repos.d": {"rhel9.repo"}},
			cacheRootDir:  "dnf",
			cacheDirs:     []string{},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
			},
		},
		"repo found, cache dir lacks -rpms- pattern": {
			dnf4ReposDirs: []string{"yum.repos.d"},
			repoDirFiles:  map[string][]string{"yum.repos.d": {"rhel9.repo"}},
			cacheRootDir:  "dnf",
			cacheDirs:     []string{"some-other-dir"},
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
			},
		},
		"no repo files and no cache layout": {
			dnf4ReposDirs: []string{"yum.repos.d"},
			repoDirFiles:  map[string][]string{"yum.repos.d": {}},
			cacheRootDir:  "dnf",
			cacheDirs:     []string{},
			expectedFlags: nil,
			expectedErrs:  []string{"no .repo files found"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cacheRoot := tt.cacheRootDir
			if cacheRoot == "" {
				cacheRoot = "dnf"
			}
			cacheDir := filepath.Join(tmpDir, cacheRoot)
			require.NoError(t, os.MkdirAll(cacheDir, 0755))

			buildDirPaths := func(dirs []string) []string {
				paths := make([]string, 0, len(dirs))
				for _, dir := range dirs {
					dirPath := filepath.Join(tmpDir, dir)
					paths = append(paths, dirPath)
					if files, ok := tt.repoDirFiles[dir]; ok {
						require.NoError(t, os.MkdirAll(dirPath, 0755))
						for _, filename := range files {
							require.NoError(t, os.WriteFile(filepath.Join(dirPath, filename), []byte("[repo]\nname=test"), 0644))
						}
					}
				}
				return paths
			}
			dnf4ReposPaths := buildDirPaths(tt.dnf4ReposDirs)
			dnf5ReposPaths := buildDirPaths(tt.dnf5ReposDirs)

			for _, dirname := range tt.cacheDirs {
				require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, dirname), 0755))
			}

			dnf4CacheDir, dnf5CacheDir := cacheDir, ""
			if tt.isDNF5Cache {
				dnf4CacheDir, dnf5CacheDir = "", cacheDir
			}
			flags, err := discoverDnfStatusFlags("", dnf4ReposPaths, dnf5ReposPaths, dnf4CacheDir, dnf5CacheDir)
			if len(tt.expectedErrs) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, expectedErr := range tt.expectedErrs {
					assert.ErrorContains(t, err, expectedErr)
				}
			}
			assert.Equal(t, tt.expectedFlags, flags)
		})
	}
}

func TestDiscoverDnfStatusFlags(t *testing.T) {
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

func TestDiscoverDnfStatusFlags_MissingDirs(t *testing.T) {
	tests := map[string]struct {
		dnf4ReposDirs []string
		repoDirFiles  map[string][]string
		cacheDir      string
		setupCache    func(string) error
		expectedFlags []v1.DnfStatusFlag
		errorContains string
	}{
		"repos dir missing, cache exists": {
			dnf4ReposDirs: []string{"nonexistent-repos"},
			repoDirFiles:  map[string][]string{},
			cacheDir:      "dnf",
			setupCache:    func(path string) error { return os.MkdirAll(path, 0755) },
			expectedFlags: nil,
			errorContains: "reading",
		},
		"repos exist, cache dir missing": {
			dnf4ReposDirs: []string{"yum.repos.d"},
			repoDirFiles: map[string][]string{
				"yum.repos.d": {"test.repo"},
			},
			cacheDir: "nonexistent-cache",
			expectedFlags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
			},
			errorContains: "reading",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cacheDirPath := filepath.Join(tmpDir, tt.cacheDir)
			if tt.setupCache != nil {
				require.NoError(t, tt.setupCache(cacheDirPath))
			}

			dnf4ReposPaths := make([]string, 0, len(tt.dnf4ReposDirs))
			for _, dir := range tt.dnf4ReposDirs {
				dirPath := filepath.Join(tmpDir, dir)
				dnf4ReposPaths = append(dnf4ReposPaths, dirPath)
				if files, ok := tt.repoDirFiles[dir]; ok {
					require.NoError(t, os.MkdirAll(dirPath, 0755))
					for _, filename := range files {
						require.NoError(t, os.WriteFile(filepath.Join(dirPath, filename), []byte("[repo]"), 0644))
					}
				}
			}

			flags, err := discoverDnfStatusFlags("", dnf4ReposPaths, nil, cacheDirPath, "")
			assert.Error(t, err)
			assert.ErrorContains(t, err, tt.errorContains)
			assert.Equal(t, tt.expectedFlags, flags)
		})
	}
}
