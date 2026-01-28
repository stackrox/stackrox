package rpm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRPMOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []PackageInfo
		wantErr  bool
	}{
		{
			name: "valid rpm output",
			input: `bash|5.3.0|2.fc43|aarch64
grep|3.11|7.fc43|aarch64
systemd|257.7|1.fc43|aarch64`,
			expected: []PackageInfo{
				{Name: "bash", Version: "5.3.0", Release: "2.fc43", Arch: "aarch64"},
				{Name: "grep", Version: "3.11", Release: "7.fc43", Arch: "aarch64"},
				{Name: "systemd", Version: "257.7", Release: "1.fc43", Arch: "aarch64"},
			},
			wantErr: false,
		},
		{
			name:  "single package",
			input: `glibc|2.42|4.fc43|aarch64`,
			expected: []PackageInfo{
				{Name: "glibc", Version: "2.42", Release: "4.fc43", Arch: "aarch64"},
			},
			wantErr: false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "whitespace only",
			input:    "   \n  \t  ",
			expected: nil,
			wantErr:  true,
		},
		{
			name: "malformed line missing field",
			input: `bash|5.3.0|2.fc43
grep|3.11|7.fc43|aarch64`,
			expected: []PackageInfo{
				{Name: "grep", Version: "3.11", Release: "7.fc43", Arch: "aarch64"},
			},
			wantErr: false,
		},
		{
			name: "mixed valid and invalid lines",
			input: `bash|5.3.0|2.fc43|aarch64
invalid-line-here
grep|3.11|7.fc43|aarch64
another|invalid
systemd|257.7|1.fc43|aarch64`,
			expected: []PackageInfo{
				{Name: "bash", Version: "5.3.0", Release: "2.fc43", Arch: "aarch64"},
				{Name: "grep", Version: "3.11", Release: "7.fc43", Arch: "aarch64"},
				{Name: "systemd", Version: "257.7", Release: "1.fc43", Arch: "aarch64"},
			},
			wantErr: false,
		},
		{
			name: "package with special characters",
			input: `python3-pip|25.1.1|16.fc43|noarch
perl-IO-Socket-SSL|2.089|2.fc43|noarch`,
			expected: []PackageInfo{
				{Name: "python3-pip", Version: "25.1.1", Release: "16.fc43", Arch: "noarch"},
				{Name: "perl-IO-Socket-SSL", Version: "2.089", Release: "2.fc43", Arch: "noarch"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRPMOutput(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPackageInfoFullVersion(t *testing.T) {
	tests := []struct {
		name     string
		pkg      PackageInfo
		expected string
	}{
		{
			name: "standard package",
			pkg: PackageInfo{
				Name:    "bash",
				Version: "5.3.0",
				Release: "2.fc43",
				Arch:    "aarch64",
			},
			expected: "5.3.0-2.fc43",
		},
		{
			name: "package with complex version",
			pkg: PackageInfo{
				Name:    "kernel",
				Version: "6.11.0",
				Release: "0.rc7.20240903git6f2e1103c34.56.fc43",
				Arch:    "aarch64",
			},
			expected: "6.11.0-0.rc7.20240903git6f2e1103c34.56.fc43",
		},
		{
			name: "simple version",
			pkg: PackageInfo{
				Name:    "simple",
				Version: "1.0",
				Release: "1",
				Arch:    "noarch",
			},
			expected: "1.0-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pkg.FullVersion()
			assert.Equal(t, tt.expected, result)
		})
	}
}
