package vsock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverOSAndVersion(t *testing.T) {
	tests := []struct {
		name            string
		osRelease       string
		expectedOS      v1.DetectedOS
		expectedVersion string
	}{
		{
			name: "RHEL 8.10",
			osRelease: `NAME="Red Hat Enterprise Linux"
VERSION="8.10 (Ootpa)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="8.10"
PLATFORM_ID="platform:el8"
PRETTY_NAME="Red Hat Enterprise Linux 8.10 (Ootpa)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:redhat:enterprise_linux:8::baseos"
HOME_URL="https://www.redhat.com/"
DOCUMENTATION_URL="https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8"
BUG_REPORT_URL="https://issues.redhat.com/"

REDHAT_BUGZILLA_PRODUCT="Red Hat Enterprise Linux 8"
REDHAT_BUGZILLA_PRODUCT_VERSION=8.10
REDHAT_SUPPORT_PRODUCT="Red Hat Enterprise Linux"
REDHAT_SUPPORT_PRODUCT_VERSION="8.10"`,
			expectedOS:      v1.DetectedOS_RHEL,
			expectedVersion: "8.10",
		},
		{
			name: "RHEL 9",
			osRelease: `ID="rhel"
VERSION_ID="9.2"`,
			expectedOS:      v1.DetectedOS_RHEL,
			expectedVersion: "9.2",
		},
		{
			name: "Non-RHEL OS",
			osRelease: `ID="debian"
VERSION_ID="12"`,
			expectedOS:      v1.DetectedOS_UNKNOWN,
			expectedVersion: "debian 12",
		},
		{
			name:            "ID field missing but VERSION_ID is present",
			osRelease:       `VERSION_ID="12"`,
			expectedOS:      v1.DetectedOS_UNKNOWN,
			expectedVersion: "unknown-OS 12",
		},
		{
			name:            "ID and VERSION_ID fields missing",
			osRelease:       ``,
			expectedOS:      v1.DetectedOS_UNKNOWN,
			expectedVersion: "",
		},
		{
			name:            "Unknown OS with version only",
			osRelease:       `VERSION_ID="10"`,
			expectedOS:      v1.DetectedOS_UNKNOWN,
			expectedVersion: "unknown-OS 10",
		},
		{
			name:            "Missing VERSION_ID",
			osRelease:       `ID="rhel"`,
			expectedOS:      v1.DetectedOS_RHEL,
			expectedVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testOsReleasePath := filepath.Join(tmpDir, "os-release")

			err := os.WriteFile(testOsReleasePath, []byte(tt.osRelease), 0644)
			require.NoError(t, err)

			// Use a testable version that accepts a path
			detectedOS, osVersion, err := discoverOSAndVersionWithPath(testOsReleasePath)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOS, detectedOS)
			assert.Equal(t, tt.expectedVersion, osVersion)
		})
	}
}

func TestDiscoverOSAndVersion_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "nonexistent")

	detectedOS, osVersion, err := discoverOSAndVersionWithPath(missingPath)
	assert.Error(t, err)
	assert.Equal(t, v1.DetectedOS_UNKNOWN, detectedOS)
	assert.Equal(t, "", osVersion)
}

func TestDiscoverOSAndVersion_MalformedOSRelease(t *testing.T) {
	tmpDir := t.TempDir()
	testOsReleasePath := filepath.Join(tmpDir, "os-release")

	err := os.WriteFile(testOsReleasePath, []byte("ID=rhel\nMALFORMED_LINE"), 0644)
	require.NoError(t, err)

	detectedOS, osVersion, err := discoverOSAndVersionWithPath(testOsReleasePath)
	assert.Error(t, err)
	assert.Equal(t, v1.DetectedOS_UNKNOWN, detectedOS)
	assert.Equal(t, "", osVersion)
}

func TestDiscoverDnfRepoFilePresence(t *testing.T) {
	tests := map[string]struct {
		setup            func(t *testing.T) (string, []string)
		expectedFound    bool
		expectedErrParts []string
	}{
		"should return true when repo file exists in reposdir": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				reposDirPath := "/etc/yum.repos.d"
				reposPath := hostPathFor(hostPath, reposDirPath)
				require.NoError(t, os.MkdirAll(reposPath, 0750))
				repoFilePath := filepath.Join(reposPath, "test.repo")
				require.NoError(t, os.WriteFile(repoFilePath, []byte("content"), 0600))
				return hostPath, []string{reposDirPath}
			},
			expectedFound: true,
		},
		"should return error when all reposdirs are unreadable": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				return hostPath, []string{"/etc/yum.repos.d", "/etc/yum/repos.d"}
			},
			expectedFound:    false,
			expectedErrParts: []string{"reading", "no such file or directory"},
		},
		"should return error when reposdirs are readable but no repo files exist": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				reposDirPaths := []string{"/etc/yum.repos.d", "/etc/yum/repos.d"}
				for _, reposDirPath := range reposDirPaths {
					reposPath := hostPathFor(hostPath, reposDirPath)
					require.NoError(t, os.MkdirAll(reposPath, 0750))
					require.NoError(t, os.WriteFile(filepath.Join(reposPath, "not-a-repo.txt"), []byte("content"), 0600))
				}
				return hostPath, reposDirPaths
			},
			expectedFound:    false,
			expectedErrParts: []string{"no .repo files found"},
		},
		"should return true when repo file exists even if other reposdir is missing": {
			setup: func(t *testing.T) (string, []string) {
				hostPath := t.TempDir()
				reposDirPaths := []string{"/etc/yum.repos.d", "/etc/yum/repos.d"}
				reposPath := hostPathFor(hostPath, reposDirPaths[0])
				require.NoError(t, os.MkdirAll(reposPath, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(reposPath, "example.repo"), []byte("content"), 0600))
				return hostPath, reposDirPaths
			},
			expectedFound: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hostPath, reposDirPaths := tt.setup(t)
			found, err := discoverDnfRepoFilePresence(hostPath, reposDirPaths)
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

func TestDiscoverDnfCacheRepoDirPresence(t *testing.T) {
	tests := map[string]struct {
		setup            func(t *testing.T) (string, string)
		expectedFound    bool
		expectedErrParts []string
	}{
		"should return true when cache dir contains repo directory": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				cacheDirPath := "/var/cache/dnf"
				cachePath := hostPathFor(hostPath, cacheDirPath)
				require.NoError(t, os.MkdirAll(filepath.Join(cachePath, "rhel-9-for-x86_64-appstream-rpms-123"), 0750))
				return hostPath, cacheDirPath
			},
			expectedFound: true,
		},
		"should return false when cache dir has no repo directories": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				cacheDirPath := "/var/cache/dnf"
				cachePath := hostPathFor(hostPath, cacheDirPath)
				require.NoError(t, os.MkdirAll(filepath.Join(cachePath, "some-other-dir"), 0750))
				return hostPath, cacheDirPath
			},
			expectedFound: false,
		},
		"should return error when cache dir is missing": {
			setup: func(t *testing.T) (string, string) {
				hostPath := t.TempDir()
				return hostPath, "/var/cache/dnf"
			},
			expectedFound:    false,
			expectedErrParts: []string{"unsupported OS detected: missing"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hostPath, cacheDirPath := tt.setup(t)
			found, err := discoverDnfCacheRepoDirPresence(hostPath, cacheDirPath)
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

func TestParseOSRelease_QuotedValues(t *testing.T) {
	input := strings.NewReader(`# comment
ID='rhel'
VERSION_ID="9.4"
NAME="Red Hat \$NAME"
`)

	values, err := parseOSRelease(input)
	require.NoError(t, err)
	assert.Equal(t, "rhel", values["ID"])
	assert.Equal(t, "9.4", values["VERSION_ID"])
	assert.Equal(t, "Red Hat $NAME", values["NAME"])
}

func TestHostPathFor(t *testing.T) {
	tests := []struct {
		name     string
		hostPath string
		path     string
		expected string
	}{
		{
			name:     "Empty host path uses original path",
			hostPath: "",
			path:     "/etc/os-release",
			expected: "/etc/os-release",
		},
		{
			name:     "Root host path uses original path",
			hostPath: "/",
			path:     "/etc/os-release",
			expected: "/etc/os-release",
		},
		{
			name:     "Prefix host path joins with absolute path",
			hostPath: "/host",
			path:     "/etc/os-release",
			expected: "/host/etc/os-release",
		},
		{
			name:     "Prefix host path joins with nested path",
			hostPath: "/host/rootfs",
			path:     "/var/cache/dnf",
			expected: "/host/rootfs/var/cache/dnf",
		},
		{
			name:     "Cleaned path removes dot segments",
			hostPath: "/root/../host",
			path:     "/var/lib/../cache//dnf/",
			expected: "/host/var/cache/dnf",
		},
		{
			name:     "Cleaned path collapses extra slashes",
			hostPath: "/host//",
			path:     "/etc/os-release",
			expected: "/host/etc/os-release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hostPathFor(tt.hostPath, tt.path))
		})
	}
}

func TestDiscoverActivationStatus(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		expectedStatus v1.ActivationStatus
	}{
		{
			name: "Activated - single pair",
			files: []string{
				"3341241341658386286-key.pem",
				"3341241341658386286.pem",
			},
			expectedStatus: v1.ActivationStatus_ACTIVE,
		},
		{
			name: "Activated - multiple pairs",
			files: []string{
				"111-key.pem",
				"111.pem",
				"222-key.pem",
				"222.pem",
			},
			expectedStatus: v1.ActivationStatus_ACTIVE,
		},
		{
			name:           "Unactivated - empty directory",
			files:          []string{},
			expectedStatus: v1.ActivationStatus_INACTIVE,
		},
		{
			name: "Unactivated - only key file",
			files: []string{
				"3341241341658386286-key.pem",
			},
			expectedStatus: v1.ActivationStatus_INACTIVE,
		},
		{
			name: "Unactivated - only cert file",
			files: []string{
				"3341241341658386286.pem",
			},
			expectedStatus: v1.ActivationStatus_INACTIVE,
		},
		{
			name: "Unactivated - mismatched names",
			files: []string{
				"111-key.pem",
				"222.pem",
			},
			expectedStatus: v1.ActivationStatus_INACTIVE,
		},
		{
			name: "Unactivated - other files",
			files: []string{
				"some-other-file.txt",
			},
			expectedStatus: v1.ActivationStatus_INACTIVE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for _, filename := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				err := os.WriteFile(filePath, []byte("test content"), 0644)
				require.NoError(t, err)
			}

			activationStatus, err := discoverActivationStatusWithPath(tmpDir)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, activationStatus)
		})
	}
}

func TestDiscoverActivationStatus_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "nonexistent")

	activationStatus, err := discoverActivationStatusWithPath(missingPath)
	assert.Error(t, err)
	assert.Equal(t, v1.ActivationStatus_ACTIVATION_UNSPECIFIED, activationStatus)
}

func TestDiscoverDnfMetadataStatus(t *testing.T) {
	tests := map[string]struct {
		reposDirs      []string
		repoDirFiles   map[string][]string
		cacheDirs      []string
		expectedStatus v1.DnfMetadataStatus
		expectedErrs   []string
	}{
		"should report available when repo file and cache dir exist": {
			reposDirs:      []string{"yum.repos.d"},
			repoDirFiles:   map[string][]string{"yum.repos.d": {"rhel9.repo"}},
			cacheDirs:      []string{"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476"},
			expectedStatus: v1.DnfMetadataStatus_AVAILABLE,
		},
		"should report available when repo file is in second reposdir": {
			reposDirs: []string{"yum.repos.d", "yum/repos.d"},
			repoDirFiles: map[string][]string{
				"yum.repos.d": {},
				"yum/repos.d": {"rhel9.repo"},
			},
			cacheDirs:      []string{"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476"},
			expectedStatus: v1.DnfMetadataStatus_AVAILABLE,
		},
		"should report available with multiple repo files and cache dirs": {
			reposDirs: []string{"yum.repos.d"},
			repoDirFiles: map[string][]string{
				"yum.repos.d": {"baseos.repo", "appstream.repo"},
			},
			cacheDirs: []string{
				"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476",
				"rhel-9-for-x86_64-baseos-rpms-a2cdae14f4ed6c20",
			},
			expectedStatus: v1.DnfMetadataStatus_AVAILABLE,
		},
		"should report unavailable when no repo files exist": {
			reposDirs:      []string{"yum.repos.d"},
			repoDirFiles:   map[string][]string{"yum.repos.d": {}},
			cacheDirs:      []string{"rhel-9-for-x86_64-appstream-rpms-3dc6dc0880df5476"},
			expectedStatus: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
			expectedErrs:   []string{"no .repo files found"},
		},
		"should report unavailable when no cache dirs exist": {
			reposDirs:      []string{"yum.repos.d"},
			repoDirFiles:   map[string][]string{"yum.repos.d": {"rhel9.repo"}},
			cacheDirs:      []string{},
			expectedStatus: v1.DnfMetadataStatus_UNAVAILABLE,
		},
		"should report unavailable when cache dir lacks -rpms- pattern": {
			reposDirs:      []string{"yum.repos.d"},
			repoDirFiles:   map[string][]string{"yum.repos.d": {"rhel9.repo"}},
			cacheDirs:      []string{"some-other-dir"},
			expectedStatus: v1.DnfMetadataStatus_UNAVAILABLE,
		},
		"should report unavailable for empty directories": {
			reposDirs:      []string{"yum.repos.d"},
			repoDirFiles:   map[string][]string{"yum.repos.d": {}},
			cacheDirs:      []string{},
			expectedStatus: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
			expectedErrs:   []string{"no .repo files found"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cacheDir := filepath.Join(tmpDir, "dnf")
			require.NoError(t, os.MkdirAll(cacheDir, 0755))

			reposDirPaths := make([]string, 0, len(tt.reposDirs))
			for _, dir := range tt.reposDirs {
				dirPath := filepath.Join(tmpDir, dir)
				reposDirPaths = append(reposDirPaths, dirPath)
				if files, ok := tt.repoDirFiles[dir]; ok {
					require.NoError(t, os.MkdirAll(dirPath, 0755))
					for _, filename := range files {
						filePath := filepath.Join(dirPath, filename)
						err := os.WriteFile(filePath, []byte("[repo]\nname=test"), 0644)
						require.NoError(t, err)
					}
				}
			}

			for _, dirname := range tt.cacheDirs {
				dirPath := filepath.Join(cacheDir, dirname)
				err := os.MkdirAll(dirPath, 0755)
				require.NoError(t, err)
			}

			dnfStatus, err := discoverDnfMetadataStatusWithPaths("", reposDirPaths, cacheDir)
			if len(tt.expectedErrs) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, expectedErr := range tt.expectedErrs {
					assert.ErrorContains(t, err, expectedErr)
				}
			}
			assert.Equal(t, tt.expectedStatus, dnfStatus)
		})
	}
}

func TestDiscoverDnfMetadataStatus_MissingDirs(t *testing.T) {
	tests := map[string]struct {
		reposDirs     []string
		repoDirFiles  map[string][]string
		cacheDir      string
		setupCache    func(string) error
		errorContains string
	}{
		"should return error when repos dir is missing": {
			reposDirs:     []string{"nonexistent-repos"},
			repoDirFiles:  map[string][]string{},
			cacheDir:      "dnf",
			setupCache:    func(path string) error { return os.MkdirAll(path, 0755) },
			errorContains: "reading",
		},
		"should return error when cache dir is missing": {
			reposDirs: []string{"yum.repos.d"},
			repoDirFiles: map[string][]string{
				"yum.repos.d": {"test.repo"},
			},
			cacheDir:      "nonexistent-cache",
			setupCache:    nil,
			errorContains: "unsupported OS detected: missing",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cacheDirPath := filepath.Join(tmpDir, tt.cacheDir)
			if tt.setupCache != nil {
				require.NoError(t, tt.setupCache(cacheDirPath))
			}

			reposDirPaths := make([]string, 0, len(tt.reposDirs))
			for _, dir := range tt.reposDirs {
				dirPath := filepath.Join(tmpDir, dir)
				reposDirPaths = append(reposDirPaths, dirPath)
				if files, ok := tt.repoDirFiles[dir]; ok {
					require.NoError(t, os.MkdirAll(dirPath, 0755))
					for _, filename := range files {
						filePath := filepath.Join(dirPath, filename)
						require.NoError(t, os.WriteFile(filePath, []byte("[repo]"), 0644))
					}
				}
			}

			dnfStatus, err := discoverDnfMetadataStatusWithPaths("", reposDirPaths, cacheDirPath)
			assert.Error(t, err)
			assert.ErrorContains(t, err, tt.errorContains)
			assert.Equal(t, v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED, dnfStatus)
		})
	}
}
