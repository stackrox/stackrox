package cpe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertCPE22to23(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard CPE 2.2",
			input:    "cpe:/o:redhat:enterprise_linux:9",
			expected: "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*",
		},
		{
			name:     "Fedora CPE 2.2",
			input:    "cpe:/o:fedoraproject:fedora:43",
			expected: "cpe:2.3:o:fedoraproject:fedora:43:*:*:*:*:*:*:*",
		},
		{
			name:     "application CPE",
			input:    "cpe:/a:apache:httpd:2.4.41",
			expected: "cpe:2.3:a:apache:httpd:2.4.41:*:*:*:*:*:*:*",
		},
		{
			name:     "hardware CPE",
			input:    "cpe:/h:intel:core_i7:8700k",
			expected: "cpe:2.3:h:intel:core_i7:8700k:*:*:*:*:*:*:*",
		},
		{
			name:     "already CPE 2.3 format",
			input:    "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*",
			expected: "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*",
		},
		{
			name:     "malformed CPE missing parts",
			input:    "cpe:/o:redhat",
			expected: "cpe:/o:redhat",
		},
		{
			name:     "not a CPE",
			input:    "not-a-cpe-string",
			expected: "not-a-cpe-string",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertCPE22to23(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseOSReleaseFile(t *testing.T) {
	// Create a temporary os-release file for testing
	tempDir := t.TempDir()
	osReleaseFile := filepath.Join(tempDir, "os-release")

	osReleaseContent := `NAME="Red Hat Enterprise Linux"
VERSION="9.4 (Plow)"
ID=rhel
VERSION_ID=9.4
PLATFORM_ID="platform:el9"
PRETTY_NAME="Red Hat Enterprise Linux 9.4 (Plow)"
ANSI_COLOR="0;31"
LOGO=fedora-logo-icon
CPE_NAME="cpe:/o:redhat:enterprise_linux:9::baseos"
HOME_URL="https://www.redhat.com/"
DOCUMENTATION_URL="https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9"
BUG_REPORT_URL="https://bugzilla.redhat.com/"
REDHAT_BUGZILLA_PRODUCT="Red Hat Enterprise Linux 9"
REDHAT_BUGZILLA_PRODUCT_VERSION=9.4
REDHAT_SUPPORT_PRODUCT="Red Hat Enterprise Linux"
REDHAT_SUPPORT_PRODUCT_VERSION="9.4"
`

	err := os.WriteFile(osReleaseFile, []byte(osReleaseContent), 0644)
	require.NoError(t, err)

	// Test parsing
	result, err := parseOSReleaseFile(osReleaseFile)
	require.NoError(t, err)

	expected := map[string]string{
		"NAME":                            "Red Hat Enterprise Linux",
		"VERSION":                         "9.4 (Plow)",
		"ID":                              "rhel",
		"VERSION_ID":                      "9.4",
		"PLATFORM_ID":                     "platform:el9",
		"PRETTY_NAME":                     "Red Hat Enterprise Linux 9.4 (Plow)",
		"ANSI_COLOR":                      "0;31",
		"LOGO":                            "fedora-logo-icon",
		"CPE_NAME":                        "cpe:/o:redhat:enterprise_linux:9::baseos",
		"HOME_URL":                        "https://www.redhat.com/",
		"DOCUMENTATION_URL":               "https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9",
		"BUG_REPORT_URL":                  "https://bugzilla.redhat.com/",
		"REDHAT_BUGZILLA_PRODUCT":         "Red Hat Enterprise Linux 9",
		"REDHAT_BUGZILLA_PRODUCT_VERSION": "9.4",
		"REDHAT_SUPPORT_PRODUCT":          "Red Hat Enterprise Linux",
		"REDHAT_SUPPORT_PRODUCT_VERSION":  "9.4",
	}

	assert.Equal(t, expected, result)
}

func TestParseOSReleaseFile_WithComments(t *testing.T) {
	// Create a temporary os-release file with comments and empty lines
	tempDir := t.TempDir()
	osReleaseFile := filepath.Join(tempDir, "os-release")

	osReleaseContent := `# This is a comment
NAME="Test Linux"
# Another comment
ID=testlinux

VERSION_ID=1.0
# Comment with equals sign = test
PRETTY_NAME="Test Linux 1.0"
`

	err := os.WriteFile(osReleaseFile, []byte(osReleaseContent), 0644)
	require.NoError(t, err)

	result, err := parseOSReleaseFile(osReleaseFile)
	require.NoError(t, err)

	expected := map[string]string{
		"NAME":        "Test Linux",
		"ID":          "testlinux",
		"VERSION_ID":  "1.0",
		"PRETTY_NAME": "Test Linux 1.0",
	}

	assert.Equal(t, expected, result)
}

func TestParseOSReleaseFile_NonExistent(t *testing.T) {
	_, err := parseOSReleaseFile("/nonexistent/file")
	assert.Error(t, err)
}

func TestParseOSReleaseFile_QuotedValues(t *testing.T) {
	tempDir := t.TempDir()
	osReleaseFile := filepath.Join(tempDir, "os-release")

	osReleaseContent := `NAME="Double Quoted"
ID=unquoted
PRETTY_NAME='Single Quoted'
VERSION="Quoted with spaces and symbols !@#"
`

	err := os.WriteFile(osReleaseFile, []byte(osReleaseContent), 0644)
	require.NoError(t, err)

	result, err := parseOSReleaseFile(osReleaseFile)
	require.NoError(t, err)

	expected := map[string]string{
		"NAME":        "Double Quoted",
		"ID":          "unquoted",
		"PRETTY_NAME": "'Single Quoted'", // Only double quotes are stripped
		"VERSION":     "Quoted with spaces and symbols !@#",
	}

	assert.Equal(t, expected, result)
}
