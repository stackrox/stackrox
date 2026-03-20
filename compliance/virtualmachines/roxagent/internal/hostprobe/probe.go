package hostprobe

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// OSReleasePath is the canonical location of os-release on Linux hosts.
	OSReleasePath = "/etc/os-release"
	// EntitlementDirPath is the default RHSM entitlement certificate directory.
	EntitlementDirPath = "/etc/pki/entitlement"
	// YumReposDirPath is the default directory containing yum/dnf .repo files.
	YumReposDirPath = "/etc/yum.repos.d"
	// DNF5ReposDirPath is the dnf5-specific repos.d directory.
	// https://github.com/rpm-software-management/dnf5/blob/185eaef1e0ad663bdb827a2179ab1df574a27d88/include/libdnf5/conf/const.hpp
	DNF5ReposDirPath = "/usr/share/dnf5/repos.d"

	// DNF4HistoryDBPath is the dnf4 transaction history SQLite database path.
	DNF4HistoryDBPath = "/var/lib/dnf/history.sqlite"
	// DNF5HistoryDBPath is the dnf5/libdnf5 transaction history SQLite database path.
	DNF5HistoryDBPath = "/usr/lib/sysimage/libdnf5/transaction_history.sqlite"

	// DNF4CacheDirPath is the default dnf4 package cache directory.
	DNF4CacheDirPath = "/var/cache/dnf"
	// DNF5CacheDirPath is the default dnf5/libdnf5 package cache directory.
	DNF5CacheDirPath = "/var/cache/libdnf5"

	entitlementKeySuffix  = "-key.pem"
	entitlementCertSuffix = ".pem"
)

// DNF4ReposDirs are the default DNF4 reposdir locations.
// https://github.com/rpm-software-management/libdnf/blob/53839f5bd88f378e57a1f1671b3db48d29984e24/libdnf/conf/ConfigMain.cpp
var DNF4ReposDirs = []string{
	"/etc/yum.repos.d",
	"/etc/yum/repos.d",
	"/etc/distro.repos.d",
}

type DNFVersion int

const (
	// DNFVersionUnknown indicates no known dnf history database was detected.
	DNFVersionUnknown DNFVersion = iota
	// DNFVersion4 indicates dnf4 transaction history was detected.
	DNFVersion4
	// DNFVersion5 indicates dnf5/libdnf5 transaction history was detected.
	DNFVersion5
)

// HostPathFor converts an absolute system path into the corresponding path under hostPath.
func HostPathFor(hostPath, path string) string {
	if hostPath == "" || hostPath == string(os.PathSeparator) {
		return path
	}
	// This join+clean approach is safe (no escape from hostPath) only when
	// the input path is absolute (e.g., "/etc/os-release"). For example,
	// hostPath="/host", path="/../etc/os-release" would clean to "/etc/os-release".
	trimmedPath := strings.TrimPrefix(path, string(os.PathSeparator))
	return filepath.Clean(filepath.Join(hostPath, trimmedPath))
}

// DetectDNFVersion returns the detected dnf major version from history DB paths.
func DetectDNFVersion(hostPath string) DNFVersion {
	if _, err := os.Stat(HostPathFor(hostPath, DNF5HistoryDBPath)); err == nil {
		return DNFVersion5
	}
	if _, err := os.Stat(HostPathFor(hostPath, DNF4HistoryDBPath)); err == nil {
		return DNFVersion4
	}
	return DNFVersionUnknown
}

// HasEntitlementCertKeyPair reports whether a matching entitlement cert/key pair exists.
func HasEntitlementCertKeyPair(hostPath string) (bool, error) {
	return hasEntitlementCertKeyPairAtPath(HostPathFor(hostPath, EntitlementDirPath))
}

// hasEntitlementCertKeyPairAtPath checks a directory for matching entitlement cert/key pairs.
// The entries returned by os.ReadDir are sorted by name, so a cert/key pair
// ("NNN.pem" followed by "NNN-key.pem") is typically found after checking just
// a few files. We return on the first match.
func hasEntitlementCertKeyPairAtPath(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	keyBases := make(map[string]struct{})
	certBases := make(map[string]struct{})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, entitlementKeySuffix) {
			base := strings.TrimSuffix(name, entitlementKeySuffix)
			keyBases[base] = struct{}{}
			if _, found := certBases[base]; found {
				return true, nil
			}
			continue
		}
		if strings.HasSuffix(name, entitlementCertSuffix) {
			base := strings.TrimSuffix(name, entitlementCertSuffix)
			certBases[base] = struct{}{}
			if _, found := keyBases[base]; found {
				return true, nil
			}
		}
	}

	return false, nil
}

// HasAnyRepoFile reports whether any of the given repo directory paths contain
// at least one *.repo file. If no directory contains a *.repo file, ReadDir
// failures are aggregated with [errors.Join] and returned (nil error when every
// inspected directory was readable but empty of *.repo files). Directory paths
// are interpreted relative to fsys (leading slashes are stripped).
func HasAnyRepoFile(fsys fs.FS, dirs []string) (bool, error) {
	var errs []error
	for _, dirPath := range dirs {
		rel := strings.TrimPrefix(dirPath, "/")
		entries, err := fs.ReadDir(fsys, rel)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".repo") {
				return true, nil
			}
		}
	}
	return false, errors.Join(errs...)
}
