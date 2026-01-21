package vsock

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var discoveryLog = logging.LoggerForModule()

const (
	osReleasePath         = "/etc/os-release"
	entitlementDirPath    = "/etc/pki/entitlement"
	yumReposDirPath       = "/etc/yum.repos.d"
	dnfCacheDirPath       = "/var/cache/dnf"
	entitlementKeySuffix  = "-key.pem"
	entitlementCertSuffix = ".pem"
	// osReleaseIDKey is the key name for the ID field in /etc/os-release.
	osReleaseIDKey = "ID"
	// osReleaseVersionIDKey is the key name for the VERSION_ID field in /etc/os-release.
	osReleaseVersionIDKey = "VERSION_ID"
	// rhelOSID is the value of the ID field in /etc/os-release for Red Hat Enterprise Linux.
	rhelOSID = "rhel"
)

// DiscoveredData holds the result of VM discovery operations.
type DiscoveredData struct {
	DetectedOS        v1.DetectedOS
	OSVersion         string
	ActivationStatus  v1.ActivationStatus
	DnfMetadataStatus v1.DnfMetadataStatus
}

// DiscoverVMData discovers VM metadata from the host system.
// Returns discovered data with defaults (UNKNOWN/UNSPECIFIED) if discovery fails.
func DiscoverVMData() *DiscoveredData {
	result := &DiscoveredData{
		DetectedOS:        v1.DetectedOS_UNKNOWN,
		OSVersion:         "",
		ActivationStatus:  v1.ActivationStatus_ACTIVATION_UNSPECIFIED,
		DnfMetadataStatus: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
	}

	// Discover OS and version from /etc/os-release.
	// Currently assumes RHEL systems: reads /etc/os-release, checks if ID field equals "rhel" to detect RHEL,
	// and extracts VERSION_ID field as the OS version. Falls back to UNKNOWN for non-RHEL systems.
	// This behavior is based on assumptions about /etc/os-release format and RHEL-specific values.
	// Future improvements may include support for other OS types and more robust version detection.
	detectedOS, osVersion, err := discoverOSAndVersionWithPath(osReleasePath)
	if err != nil {
		discoveryLog.Warnf("Failed to discover OS and version: %v", err)
	} else {
		result.DetectedOS = detectedOS
		result.OSVersion = osVersion
	}

	// Discover activation status from /etc/pki/entitlement.
	// Currently assumes RHEL entitlement certificates: checks for matching cert/key pairs by filename
	// (e.g., "123-key.pem" and "123.pem"). System is considered ACTIVATED if at least one matching pair exists,
	// otherwise INACTIVE. This behavior is based on assumptions about RHEL entitlement certificate naming
	// conventions and file structure. Future improvements may include actual certificate validation and
	// support for other activation mechanisms.
	activationStatus, err := discoverActivationStatusWithPath(entitlementDirPath)
	if err != nil {
		discoveryLog.Warnf("Failed to discover activation status: %v", err)
	} else {
		result.ActivationStatus = activationStatus
	}

	// Discover DNF metadata status.
	// Currently assumes RHEL DNF setup: checks for both (1) at least one *.repo file in /etc/yum.repos.d
	// and (2) at least one directory in /var/cache/dnf containing "-rpms-" in its name. Metadata is
	// considered AVAILABLE only if both conditions are met. This behavior is based on assumptions about
	// RHEL repository configuration and DNF cache directory naming patterns. Future improvements may
	// include more accurate detection methods (e.g., checking actual cache contents or DNF database state).
	dnfStatus, err := discoverDnfMetadataStatusWithPaths(yumReposDirPath, dnfCacheDirPath)
	if err != nil {
		discoveryLog.Warnf("Failed to discover DNF metadata status: %v", err)
	} else {
		result.DnfMetadataStatus = dnfStatus
	}

	return result
}

// discoverOSAndVersionWithPath reads os-release from the given path and returns DetectedOS and OSVersion.
func discoverOSAndVersionWithPath(path string) (v1.DetectedOS, string, error) {
	file, err := os.Open(path)
	if err != nil {
		logPathError(path, err)
		return v1.DetectedOS_UNKNOWN, "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			discoveryLog.Warnf("Failed to close %s: %v", path, err)
		}
	}()

	osRelease := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove quotes if present
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}
		osRelease[key] = value
	}

	if err := scanner.Err(); err != nil {
		return v1.DetectedOS_UNKNOWN, "", fmt.Errorf("reading %s: %w", path, err)
	}

	// Determine DetectedOS based on ID field
	var detectedOS v1.DetectedOS
	if id, ok := osRelease[osReleaseIDKey]; ok && id == rhelOSID {
		detectedOS = v1.DetectedOS_RHEL
	} else {
		detectedOS = v1.DetectedOS_UNKNOWN
	}

	// Get OS version from VERSION_ID
	var osVersion string
	if versionID, ok := osRelease[osReleaseVersionIDKey]; ok {
		osVersion = versionID
	}

	return detectedOS, osVersion, nil
}

// discoverActivationStatusWithPath checks the given path for matching cert/key pairs.
func discoverActivationStatusWithPath(path string) (v1.ActivationStatus, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		logPathError(path, err)
		return v1.ActivationStatus_ACTIVATION_UNSPECIFIED, fmt.Errorf("reading %s: %w", path, err)
	}

	// Build maps of base names (without suffix) for keys and certs
	keyBases := make(map[string]bool)
	certBases := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, entitlementKeySuffix) {
			base := strings.TrimSuffix(name, entitlementKeySuffix)
			keyBases[base] = true
		} else if strings.HasSuffix(name, entitlementCertSuffix) {
			base := strings.TrimSuffix(name, entitlementCertSuffix)
			certBases[base] = true
		}
	}

	// Check if there's at least one matching pair
	hasPair := false
	for base := range keyBases {
		if certBases[base] {
			hasPair = true
			break
		}
	}

	if hasPair {
		return v1.ActivationStatus_ACTIVE, nil
	}
	return v1.ActivationStatus_INACTIVE, nil
}

// discoverDnfMetadataStatusWithPaths checks both repos and cache directories.
func discoverDnfMetadataStatusWithPaths(reposDirPath, cacheDirPath string) (v1.DnfMetadataStatus, error) {
	// Check for repo files in /etc/yum.repos.d
	repoEntries, err := os.ReadDir(reposDirPath)
	if err != nil {
		logPathError(reposDirPath, err)
		return v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED, fmt.Errorf("reading %s: %w", reposDirPath, err)
	}

	hasRepoFile := false
	for _, entry := range repoEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".repo") {
			hasRepoFile = true
			break
		}
	}

	if !hasRepoFile {
		return v1.DnfMetadataStatus_UNAVAILABLE, nil
	}

	// Check for repo directories in /var/cache/dnf
	cacheEntries, err := os.ReadDir(cacheDirPath)
	if err != nil {
		logPathError(cacheDirPath, err)
		return v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED, fmt.Errorf("reading %s: %w", cacheDirPath, err)
	}

	hasRepoDir := false
	for _, entry := range cacheEntries {
		if entry.IsDir() {
			// Check if it looks like a repo directory (contains "-rpms-" pattern)
			if strings.Contains(entry.Name(), "-rpms-") {
				hasRepoDir = true
				break
			}
		}
	}

	if hasRepoFile && hasRepoDir {
		return v1.DnfMetadataStatus_AVAILABLE, nil
	}
	return v1.DnfMetadataStatus_UNAVAILABLE, nil
}

func logPathError(path string, err error) {
	if os.IsNotExist(err) {
		discoveryLog.Warnf("Unsupported OS detected: missing %s", path)
		return
	}
	discoveryLog.Warnf("Failed to read %s: %v", path, err)
}
