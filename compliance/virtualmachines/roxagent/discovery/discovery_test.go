package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
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
			hostPath := t.TempDir()
			entitlementDir := hostprobe.HostPathFor(hostPath, hostprobe.EntitlementDirPath)
			require.NoError(t, os.MkdirAll(entitlementDir, 0755))

			for _, filename := range tt.files {
				err := os.WriteFile(filepath.Join(entitlementDir, filename), []byte("test content"), 0644)
				require.NoError(t, err)
			}

			activationStatus, err := discoverActivationStatus(hostPath)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, activationStatus)
		})
	}
}

func TestDiscoverActivationStatus_MissingDir(t *testing.T) {
	hostPath := t.TempDir()

	activationStatus, err := discoverActivationStatus(hostPath)
	assert.Error(t, err)
	assert.Equal(t, v1.ActivationStatus_ACTIVATION_UNSPECIFIED, activationStatus)
}
