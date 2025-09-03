package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/agent/internal/cpe"
	"github.com/stackrox/rox/agent/internal/rpm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIndexReport(t *testing.T) {
	// Create test data
	packages := []rpm.PackageInfo{
		{Name: "bash", Version: "5.1.8", Release: "9.el9", Arch: "aarch64"},
		{Name: "glibc", Version: "2.34", Release: "83.el9", Arch: "aarch64"},
	}

	osInfo := &cpe.OSInfo{
		ID:         "rhel",
		Name:       "Red Hat Enterprise Linux",
		Version:    "9.4",
		VersionID:  "9",
		PrettyName: "Red Hat Enterprise Linux 9.4 (Plow)",
		CPEName:    "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*",
		Arch:       "aarch64",
	}

	// Build IndexReport
	indexReport, err := BuildIndexReport(packages, osInfo)
	require.NoError(t, err)
	require.NotNil(t, indexReport)

	// Test basic structure
	assert.NotEmpty(t, indexReport.VsockCid)
	assert.NotNil(t, indexReport.IndexV4)
	assert.True(t, indexReport.IndexV4.Success)
	assert.Contains(t, indexReport.IndexV4.HashId, "/v4/vm/")

	// Test contents
	contents := indexReport.IndexV4.Contents
	require.NotNil(t, contents)

	// Test packages
	assert.Len(t, contents.Packages, 2)

	// Test first package
	pkg1 := contents.Packages[0]
	assert.Equal(t, "0", pkg1.Id)
	assert.Equal(t, "bash", pkg1.Name)
	assert.Equal(t, "5.1.8-9.el9", pkg1.Version)
	assert.Equal(t, "binary", pkg1.Kind)
	assert.Equal(t, "sqlite:usr/share/rpm", pkg1.PackageDb)
	assert.Equal(t, "aarch64", pkg1.Arch)
	assert.Equal(t, "cpe:2.3:a:redhat:bash:5.1.8-9.el9:*:*:*:*:*:*:*", pkg1.Cpe)
	assert.Contains(t, pkg1.RepositoryHint, "rpm:")

	// Test source package
	require.NotNil(t, pkg1.Source)
	assert.Equal(t, "bash", pkg1.Source.Name)
	assert.Equal(t, "5.1.8-9.el9", pkg1.Source.Version)
	assert.Equal(t, "source", pkg1.Source.Kind)
	assert.Equal(t, "cpe:2.3:a:redhat:bash:5.1.8-9.el9:*:*:*:*:*:*:*", pkg1.Source.Cpe)

	// Test distributions
	assert.Len(t, contents.Distributions, 1)
	dist := contents.Distributions[0]
	assert.Equal(t, "rhel-9", dist.Id)
	assert.Equal(t, "rhel", dist.Did)
	assert.Equal(t, "Red Hat Enterprise Linux", dist.Name)
	assert.Equal(t, "9.4", dist.Version)
	assert.Equal(t, "9", dist.VersionId)
	assert.Equal(t, "aarch64", dist.Arch)
	assert.Equal(t, "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*", dist.Cpe)

	// Test repositories
	assert.Len(t, contents.Repositories, 1)
	repo := contents.Repositories[0]
	assert.Equal(t, "0", repo.Id)
	assert.Equal(t, "rhel-cpe-repository", repo.Key)
	assert.Equal(t, "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*", repo.Cpe)

	// Test environments
	assert.Len(t, contents.Environments, 2)

	// Test environment for first package
	env1, exists := contents.Environments["0"]
	require.True(t, exists)
	require.Len(t, env1.Environments, 1)
	assert.Equal(t, "sqlite:usr/share/rpm", env1.Environments[0].PackageDb)
	assert.Equal(t, []string{"0"}, env1.Environments[0].RepositoryIds)
	assert.NotEmpty(t, env1.Environments[0].IntroducedIn)
}

func TestBuildIndexReport_EmptyPackages(t *testing.T) {
	packages := []rpm.PackageInfo{}
	osInfo := &cpe.OSInfo{
		ID:        "rhel",
		Name:      "Red Hat Enterprise Linux",
		Version:   "9.4",
		VersionID: "9",
		CPEName:   "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*",
		Arch:      "aarch64",
	}

	indexReport, err := BuildIndexReport(packages, osInfo)
	require.NoError(t, err)
	require.NotNil(t, indexReport)

	contents := indexReport.IndexV4.Contents
	assert.Len(t, contents.Packages, 0)
	assert.Len(t, contents.Environments, 0)
	assert.Len(t, contents.Distributions, 1) // Distribution should still be present
	assert.Len(t, contents.Repositories, 1)  // Repository should still be present
}

func TestBuildIndexReport_FedoraOS(t *testing.T) {
	packages := []rpm.PackageInfo{
		{Name: "systemd", Version: "257.7", Release: "1.fc43", Arch: "aarch64"},
	}

	osInfo := &cpe.OSInfo{
		ID:         "fedora",
		Name:       "Fedora Linux",
		Version:    "43",
		VersionID:  "43",
		PrettyName: "Fedora Linux 43",
		CPEName:    "cpe:2.3:o:fedoraproject:fedora:43:*:*:*:*:*:*:*",
		Arch:       "aarch64",
	}

	indexReport, err := BuildIndexReport(packages, osInfo)
	require.NoError(t, err)

	// Test Fedora-specific values
	contents := indexReport.IndexV4.Contents
	pkg := contents.Packages[0]
	assert.Equal(t, "cpe:2.3:a:fedoraproject:systemd:257.7-1.fc43:*:*:*:*:*:*:*", pkg.Cpe)

	dist := contents.Distributions[0]
	assert.Equal(t, "fedora-43", dist.Id)
	assert.Equal(t, "fedora", dist.Did)
	assert.Equal(t, "Fedora Linux", dist.Name)
}

func TestGetHostname(t *testing.T) {
	// Create temporary hostname files for testing
	tempDir := t.TempDir()

	// Test /etc/hostname
	etcHostname := filepath.Join(tempDir, "hostname")
	err := os.WriteFile(etcHostname, []byte("test-hostname\n"), 0644)
	require.NoError(t, err)

	// Test /proc/sys/kernel/hostname
	procDir := filepath.Join(tempDir, "proc", "sys", "kernel")
	err = os.MkdirAll(procDir, 0755)
	require.NoError(t, err)

	procHostname := filepath.Join(procDir, "hostname")
	err = os.WriteFile(procHostname, []byte("proc-hostname"), 0644)
	require.NoError(t, err)

	// Test preference order - /etc/hostname should be preferred
	// Note: This test would need the actual function to be refactored to accept
	// custom paths for testing, but demonstrates the testing approach

	// For now, just test that the function doesn't crash
	hostname, err := getHostname()
	require.NoError(t, err)
	assert.NotEmpty(t, hostname)
}

func TestReadHostnameFromFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple hostname",
			content:  "test-hostname",
			expected: "test-hostname",
		},
		{
			name:     "hostname with newline",
			content:  "test-hostname\n",
			expected: "test-hostname\n",
		},
		{
			name:     "hostname with multiple lines",
			content:  "test-hostname\nextra-line",
			expected: "test-hostname\nextra-line",
		},
		{
			name:     "empty file",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "hostname")

			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			result, err := readHostnameFromFile(testFile)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadHostnameFromFile_NonExistent(t *testing.T) {
	_, err := readHostnameFromFile("/nonexistent/file")
	assert.Error(t, err)
}
