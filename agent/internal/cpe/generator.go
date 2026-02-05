package cpe

import (
	"fmt"

	"github.com/stackrox/rox/agent/internal/rpm"
)

// GeneratePackageCPE generates a CPE 2.3 identifier for a package
// Format: cpe:2.3:a:vendor:package_name:version-release:*:*:*:*:*:*:*
func GeneratePackageCPE(pkg rpm.PackageInfo, osInfo *OSInfo) string {
	vendor := getVendorForOS(osInfo.ID)
	return fmt.Sprintf("cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*",
		vendor,
		pkg.Name,
		pkg.FullVersion(),
	)
}

// getVendorForOS returns the appropriate vendor string based on the OS ID
func getVendorForOS(osID string) string {
	switch osID {
	case "rhel":
		return "redhat"
	case "fedora":
		return "fedoraproject"
	default:
		// Default to redhat for compatibility with existing implementation
		return "redhat"
	}
}
