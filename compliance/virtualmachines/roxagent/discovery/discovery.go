package discovery

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const (
	osReleaseIDKey        = "ID"
	osReleaseVersionIDKey = "VERSION_ID"
	rhelOSID              = "rhel"
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
	detectedOS, osVersion, err := discoverOSAndVersionWithPath(hostprobe.HostPathFor(hostPath, hostprobe.OSReleasePath))
	if err != nil {
		log.Infof("Unable to discover OS and version: %v", err)
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
	activationStatus, err := discoverActivationStatus(hostPath)
	if err != nil {
		log.Infof("Observations during discovering activation status: %v", err)
	}
	// Some errors are of a warning nature, so we still set the discovery result.
	result.ActivationStatus = activationStatus

	// Discover granular DNF status flags and derive the legacy metadata status.
	dnfFlags, err := discoverDnfStatusFlags(hostPath,
		hostprobe.DNF4ReposDirs, []string{hostprobe.DNF5ReposDirPath},
		hostprobe.DNF4CacheDirPath, hostprobe.DNF5CacheDirPath,
	)
	if err != nil {
		log.Debugf("Observations during discovering DNF status flags: %v", err)
	}
	result.DnfStatus = dnfFlags
	result.DnfMetadataStatus = deriveLegacyDnfMetadataStatus(dnfFlags)

	return result
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

	var detectedOS v1.DetectedOS
	if id, ok := osRelease[osReleaseIDKey]; ok && strings.TrimSpace(id) == rhelOSID {
		detectedOS = v1.DetectedOS_RHEL
	} else {
		detectedOS = v1.DetectedOS_UNKNOWN
	}

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

// discoverActivationStatus checks the host for matching entitlement cert/key pairs.
func discoverActivationStatus(hostPath string) (v1.ActivationStatus, error) {
	hasPair, err := hostprobe.HasEntitlementCertKeyPair(hostPath)
	if err != nil {
		return v1.ActivationStatus_ACTIVATION_UNSPECIFIED, err
	}
	if hasPair {
		return v1.ActivationStatus_ACTIVE, nil
	}
	return v1.ActivationStatus_INACTIVE, nil
}
