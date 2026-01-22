package vsock

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/set"
)

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

// DiscoverVMData discovers VM metadata from the host system.
// Returns discovered data with defaults (UNKNOWN/UNSPECIFIED) if discovery fails.
func DiscoverVMData(hostPath string) *v1.DiscoveredData {
	result := &v1.DiscoveredData{}

	// Discover OS and version from /etc/os-release.
	// Currently assumes RHEL systems: reads /etc/os-release, checks if ID field equals "rhel" to detect RHEL,
	// and extracts VERSION_ID field as the OS version. Falls back to UNKNOWN for non-RHEL systems.
	// This behavior is based on assumptions about /etc/os-release format and RHEL-specific values.
	// Future improvements may include support for other OS types and more robust version detection.
	detectedOS, osVersion, err := discoverOSAndVersionWithPath(hostPathFor(hostPath, osReleasePath))
	if err != nil {
		log.Warnf("Failed to discover OS and version: %v", err)
	} else {
		result.DetectedOs = detectedOS
		result.OsVersion = osVersion
	}

	// Discover activation status from /etc/pki/entitlement.
	// Currently assumes RHEL entitlement certificates: checks for matching cert/key pairs by filename
	// (e.g., "123-key.pem" and "123.pem"). System is considered ACTIVATED if at least one matching pair exists,
	// otherwise INACTIVE. This behavior is based on assumptions about RHEL entitlement certificate naming
	// conventions and file structure. Future improvements may include actual certificate validation and
	// support for other activation mechanisms.
	activationStatus, err := discoverActivationStatusWithPath(hostPathFor(hostPath, entitlementDirPath))
	if err != nil {
		log.Warnf("Failed to discover activation status: %v", err)
	} else {
		result.ActivationStatus = activationStatus
	}

	// Discover DNF metadata status.
	// Currently assumes RHEL DNF setup: checks for both (1) at least one *.repo file in /etc/yum.repos.d
	// and (2) at least one directory in /var/cache/dnf containing "-rpms-" in its name. Metadata is
	// considered AVAILABLE only if both conditions are met. This behavior is based on assumptions about
	// RHEL repository configuration and DNF cache directory naming patterns. Future improvements may
	// include more accurate detection methods (e.g., checking actual cache contents or DNF database state).
	dnfStatus, err := discoverDnfMetadataStatusWithPaths(
		hostPathFor(hostPath, yumReposDirPath),
		hostPathFor(hostPath, dnfCacheDirPath),
	)
	if err != nil {
		log.Warnf("Failed to discover DNF metadata status: %v", err)
	} else {
		result.DnfMetadataStatus = dnfStatus
	}

	return result
}

func hostPathFor(hostPath, path string) string {
	if hostPath == "" || hostPath == string(os.PathSeparator) {
		return path
	}
	// This join+clean approach is safe (no escape from hostPath) only when
	// the input path is absolute (e.g., "/etc/os-release"). For example,
	// hostPath="/host", path="/../etc/os-release" would clean to "/etc/os-release".
	trimmedPath := strings.TrimPrefix(path, string(os.PathSeparator))
	return filepath.Clean(filepath.Join(hostPath, trimmedPath))
}

// discoverOSAndVersionWithPath reads os-release from the given path and returns DetectedOS and OSVersion.
func discoverOSAndVersionWithPath(path string) (v1.DetectedOS, string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return v1.DetectedOS_UNKNOWN, "", fmt.Errorf("unsupported OS detected: missing %s: %w", path, err)
		}
		return v1.DetectedOS_UNKNOWN, "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warnf("Failed to close %s: %v", path, err)
		}
	}()

	osRelease, err := parseOSRelease(file)
	if err != nil {
		return v1.DetectedOS_UNKNOWN, "", fmt.Errorf("reading %s: %w", path, err)
	}

	// Determine DetectedOS based on ID field
	var detectedOS v1.DetectedOS
	if id, ok := osRelease[osReleaseIDKey]; ok && strings.TrimSpace(id) == rhelOSID {
		detectedOS = v1.DetectedOS_RHEL
	} else {
		detectedOS = v1.DetectedOS_UNKNOWN
	}

	// Get OS version from VERSION_ID
	var osVersion string
	if versionID, ok := osRelease[osReleaseVersionIDKey]; ok {
		if detectedOS != v1.DetectedOS_UNKNOWN {
			return detectedOS, strings.TrimSpace(versionID), nil
		}
		// For non-RHEL systems, store the name of the OS (ID) and version (VERSION_ID) together.
		// The version field is only informative and used for debugging in case of problems with scanning;
		// we want to know which OS and version caused a potential issue.
		osID := strings.TrimSpace(osRelease[osReleaseIDKey])
		if osID == "" {
			osID = "unknown-OS"
		}
		osVersion = fmt.Sprintf("%s %s", osID, versionID)
	}

	return detectedOS, osVersion, nil
}

// parseOSRelease parses /etc/os-release key-value pairs.
//
// We copy ClairCore's os-release parser instead of importing it to avoid pulling
// in heavy scanner/indexer dependencies into roxagent. As Rob Pike put it,
// "a little bit of copying is better than a little bit of dependency."
//
// Source (copied, adapted to our usage):
// https://github.com/quay/claircore/blob/9f69181a1555935c8840a9191c91567e55b9cf0c/osrelease/scanner.go
func parseOSRelease(r io.Reader) (map[string]string, error) {
	osRelease := make(map[string]string)
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		b := bytes.TrimSpace(scanner.Bytes())
		switch {
		case len(b) == 0:
			continue
		case b[0] == '#':
			continue
		}
		if !bytes.ContainsRune(b, '=') {
			return nil, fmt.Errorf("osrelease: malformed line %q", scanner.Text())
		}
		key, value, _ := strings.Cut(string(b), "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch {
		case len(value) == 0:
		case value[0] == '\'':
			value = strings.TrimFunc(value, func(r rune) bool { return r == '\'' })
			value = strings.ReplaceAll(value, `'\''`, `'`)
		case value[0] == '"':
			value = strings.TrimFunc(value, func(r rune) bool { return r == '"' })
			value = osReleaseDQReplacer.Replace(value)
		default:
		}
		osRelease[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return osRelease, nil
}

var osReleaseDQReplacer = strings.NewReplacer(
	"\\`", "`",
	`\\`, `\`,
	`\"`, `"`,
	`\$`, `$`,
)

// discoverActivationStatusWithPath checks the given path for matching cert/key pairs.
func discoverActivationStatusWithPath(path string) (v1.ActivationStatus, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return v1.ActivationStatus_ACTIVATION_UNSPECIFIED, fmt.Errorf("unsupported OS detected: missing %s: %w", path, err)
		}
		return v1.ActivationStatus_ACTIVATION_UNSPECIFIED, fmt.Errorf("reading %s: %w", path, err)
	}

	// Build sets of base names (without suffix) for keys and certs
	keyBases := set.NewStringSet()
	certBases := set.NewStringSet()

	// The `entries` are already sorted by name, so optimistically we just need to check two files.
	// We can stop when first matching pair is found.
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, entitlementKeySuffix) {
			base := strings.TrimSuffix(name, entitlementKeySuffix)
			keyBases.Add(base)
			if certBases.Contains(base) {
				return v1.ActivationStatus_ACTIVE, nil
			}
		} else if strings.HasSuffix(name, entitlementCertSuffix) {
			base := strings.TrimSuffix(name, entitlementCertSuffix)
			certBases.Add(base)
			if keyBases.Contains(base) {
				return v1.ActivationStatus_ACTIVE, nil
			}
		}
	}

	return v1.ActivationStatus_INACTIVE, nil
}

// discoverDnfMetadataStatusWithPaths checks both repos and cache directories.
func discoverDnfMetadataStatusWithPaths(reposDirPath, cacheDirPath string) (v1.DnfMetadataStatus, error) {
	// Check for repo files in /etc/yum.repos.d
	repoEntries, err := os.ReadDir(reposDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED, fmt.Errorf("unsupported OS detected: missing %s: %w", reposDirPath, err)
		}
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
		if os.IsNotExist(err) {
			return v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED, fmt.Errorf("unsupported OS detected: missing %s: %w", cacheDirPath, err)
		}
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
