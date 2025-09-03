package report

import (
	"fmt"
	"os"

	"github.com/stackrox/rox/agent/internal/cpe"
	"github.com/stackrox/rox/agent/internal/rpm"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// BuildIndexReport creates a complete IndexReport from packages and OS information
func BuildIndexReport(packages []rpm.PackageInfo, osInfo *cpe.OSInfo) (*v1.IndexReport, error) {
	// Get hostname for VSOCK CID identifier
	hostname, err := getHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// Convert packages to protobuf Package structs
	pbPackages := make([]*v4.Package, 0, len(packages))
	environments := make(map[string]*v4.Environment_List)

	for i, pkg := range packages {
		packageCPE := cpe.GeneratePackageCPE(pkg, osInfo)

		// Create source package (required by Scanner V4)
		sourcePackage := &v4.Package{
			Name:    pkg.Name,
			Version: pkg.FullVersion(),
			Kind:    "source",
			Cpe:     packageCPE,
		}

		// Create binary package
		pbPackage := &v4.Package{
			Id:             fmt.Sprintf("%d", i),
			Name:           pkg.Name,
			Version:        pkg.FullVersion(),
			Kind:           "binary",
			Source:         sourcePackage,
			PackageDb:      "sqlite:usr/share/rpm",
			RepositoryHint: fmt.Sprintf("rpm:%s:%s", pkg.Arch, pkg.FullVersion()),
			Arch:           pkg.Arch,
			Cpe:            packageCPE,
		}

		pbPackages = append(pbPackages, pbPackage)

		// Create environment mapping (required by Scanner V4)
		env := &v4.Environment{
			PackageDb:     "sqlite:usr/share/rpm",
			IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			RepositoryIds: []string{"0"},
		}
		environments[fmt.Sprintf("%d", i)] = &v4.Environment_List{
			Environments: []*v4.Environment{env},
		}
	}

	// Create distribution
	distribution := &v4.Distribution{
		Id:         fmt.Sprintf("%s-%s", osInfo.ID, osInfo.VersionID),
		Did:        osInfo.ID,
		Name:       osInfo.Name,
		Version:    osInfo.Version,
		VersionId:  osInfo.VersionID,
		Arch:       osInfo.Arch,
		Cpe:        osInfo.CPEName,
		PrettyName: osInfo.PrettyName,
	}

	// Create repository
	repository := &v4.Repository{
		Id:   "0",
		Name: fmt.Sprintf("cpe:/%s", osInfo.CPEName),
		Key:  "rhel-cpe-repository",
		Cpe:  osInfo.CPEName,
	}

	// Create Contents
	contents := &v4.Contents{
		Packages:      pbPackages,
		Distributions: []*v4.Distribution{distribution},
		Repositories:  []*v4.Repository{repository},
		Environments:  environments,
	}

	// Create Scanner V4 IndexReport
	indexV4 := &v4.IndexReport{
		HashId:   fmt.Sprintf("/v4/vm/%s", hostname),
		Success:  true,
		Contents: contents,
	}

	// Create VM IndexReport
	return &v1.IndexReport{
		VsockCid: hostname, // Use hostname as identifier
		IndexV4:  indexV4,
	}, nil
}

// getHostname returns the system hostname
func getHostname() (string, error) {
	// Try common hostname sources in order of preference
	hostnamePaths := []string{"/etc/hostname", "/proc/sys/kernel/hostname"}

	for _, path := range hostnamePaths {
		if hostname, err := readHostnameFromFile(path); err == nil && hostname != "" {
			return hostname, nil
		}
	}

	// Fallback to default
	return "no-hostname", nil
}

// readHostnameFromFile reads hostname from a file
func readHostnameFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read hostname from %s: %w", path, err)
	}
	return string(data), nil
}
