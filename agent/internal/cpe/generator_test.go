package cpe

import (
	"testing"

	"github.com/stackrox/rox/agent/internal/rpm"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePackageCPE(t *testing.T) {
	tests := []struct {
		name     string
		pkg      rpm.PackageInfo
		osInfo   *OSInfo
		expected string
	}{
		{
			name: "RHEL package",
			pkg: rpm.PackageInfo{
				Name:    "bash",
				Version: "5.1.8",
				Release: "9.el9",
				Arch:    "aarch64",
			},
			osInfo: &OSInfo{
				ID: "rhel",
			},
			expected: "cpe:2.3:a:redhat:bash:5.1.8-9.el9:*:*:*:*:*:*:*",
		},
		{
			name: "Fedora package",
			pkg: rpm.PackageInfo{
				Name:    "systemd",
				Version: "257.7",
				Release: "1.fc43",
				Arch:    "aarch64",
			},
			osInfo: &OSInfo{
				ID: "fedora",
			},
			expected: "cpe:2.3:a:fedoraproject:systemd:257.7-1.fc43:*:*:*:*:*:*:*",
		},
		{
			name: "Unknown OS defaults to redhat",
			pkg: rpm.PackageInfo{
				Name:    "glibc",
				Version: "2.42",
				Release: "4.fc43",
				Arch:    "aarch64",
			},
			osInfo: &OSInfo{
				ID: "unknown",
			},
			expected: "cpe:2.3:a:redhat:glibc:2.42-4.fc43:*:*:*:*:*:*:*",
		},
		{
			name: "Package with special characters",
			pkg: rpm.PackageInfo{
				Name:    "python3-pip",
				Version: "25.1.1",
				Release: "16.fc43",
				Arch:    "noarch",
			},
			osInfo: &OSInfo{
				ID: "fedora",
			},
			expected: "cpe:2.3:a:fedoraproject:python3-pip:25.1.1-16.fc43:*:*:*:*:*:*:*",
		},
		{
			name: "Complex version string",
			pkg: rpm.PackageInfo{
				Name:    "kernel",
				Version: "6.11.0",
				Release: "0.rc7.20240903git6f2e1103c34.56.fc43",
				Arch:    "aarch64",
			},
			osInfo: &OSInfo{
				ID: "fedora",
			},
			expected: "cpe:2.3:a:fedoraproject:kernel:6.11.0-0.rc7.20240903git6f2e1103c34.56.fc43:*:*:*:*:*:*:*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePackageCPE(tt.pkg, tt.osInfo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetVendorForOS(t *testing.T) {
	tests := []struct {
		name     string
		osID     string
		expected string
	}{
		{
			name:     "RHEL",
			osID:     "rhel",
			expected: "redhat",
		},
		{
			name:     "Fedora",
			osID:     "fedora",
			expected: "fedoraproject",
		},
		{
			name:     "CentOS defaults to redhat",
			osID:     "centos",
			expected: "redhat",
		},
		{
			name:     "Unknown OS defaults to redhat",
			osID:     "unknown",
			expected: "redhat",
		},
		{
			name:     "Empty string defaults to redhat",
			osID:     "",
			expected: "redhat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getVendorForOS(tt.osID)
			assert.Equal(t, tt.expected, result)
		})
	}
}
