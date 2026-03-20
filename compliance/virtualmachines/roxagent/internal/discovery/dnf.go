package discovery

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// discoverDnfRepoFilePresence reports whether any of the given reposdirs contains a *.repo file.
//   - Returns hasDir=true when at least one reposdir is readable.
//   - Returns hasRepo=true as soon as a *.repo file is found in any reposdir.
//   - If reposdirs are unreadable or contain no *.repo files, returns an aggregated error.
func discoverDnfRepoFilePresence(hostPath string, reposDirPaths []string) (hasDir, hasRepo bool, err error) {
	if len(reposDirPaths) == 0 {
		return false, false, errors.New("missing repository directories")
	}

	var repoDirErrs *multierror.Error
	for _, reposDirPath := range reposDirPaths {
		reposPath := hostprobe.HostPathFor(hostPath, reposDirPath)
		repoEntries, err := os.ReadDir(reposPath)
		if err != nil {
			repoDirErrs = multierror.Append(repoDirErrs, fmt.Errorf("reading %q: %w", reposPath, err))
			continue
		}
		hasDir = true

		for _, entry := range repoEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".repo") {
				return true, true, nil
			}
		}
		repoDirErrs = multierror.Append(repoDirErrs, fmt.Errorf("no .repo files found in %q", reposPath))

	}

	return hasDir, hasRepo, repoDirErrs.ErrorOrNil()
}

// discoverDnf4CacheRepoDirPresence reports whether hostPath/cacheDirPath looks like a DNF4 package cache:
// subdirectories whose names contain "-rpms-" under the cache root (e.g. /var/cache/dnf).
//
// DNF4 default layout (libdnf Const.hpp):
// https://github.com/rpm-software-management/libdnf/blob/53839f5bd88f378e57a1f1671b3db48d29984e24/libdnf/conf/Const.hpp
func discoverDnf4CacheRepoDirPresence(hostPath, cacheDirPath string) (hasDir, hasRepo bool, err error) {
	cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
	cacheEntries, err := os.ReadDir(cachePath)
	if err != nil {
		return false, false, fmt.Errorf("reading %s: %w", cachePath, err)
	}
	hasRepo = slices.ContainsFunc(cacheEntries, func(e os.DirEntry) bool {
		return e.IsDir() && strings.Contains(e.Name(), "-rpms-")
	})
	return true, hasRepo, nil
}

// discoverDnf5CacheRepoDirPresence reports whether hostPath/cacheDirPath looks like a DNF5/libdnf5 cache root
// (e.g. /var/cache/libdnf5): any subdirectory indicates repository cache layout.
//
// DNF5 defaults (libdnf5 const.hpp):
// https://github.com/rpm-software-management/dnf5/blob/185eaef1e0ad663bdb827a2179ab1df574a27d88/include/libdnf5/conf/const.hpp
func discoverDnf5CacheRepoDirPresence(hostPath, cacheDirPath string) (hasDir, hasRepo bool, err error) {
	cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
	cacheEntries, err := os.ReadDir(cachePath)
	if err != nil {
		return false, false, fmt.Errorf("reading %s: %w", cachePath, err)
	}
	hasRepo = slices.ContainsFunc(cacheEntries, func(e os.DirEntry) bool {
		return e.IsDir()
	})
	return true, hasRepo, nil
}

// deriveLegacyDnfMetadataStatus maps DnfStatusFlag values to the deprecated
// DnfMetadataStatus enum for backward compatibility with 4.10 tech-preview agents.
func deriveLegacyDnfMetadataStatus(flags []v1.DnfStatusFlag) v1.DnfMetadataStatus {
	if len(flags) == 0 {
		return v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED
	}
	hasRepo, hasCache := false, false
	for _, f := range flags {
		switch f {
		case v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND:
			hasRepo = true
		case v1.DnfStatusFlag_DNF_V4_CACHE_FOUND, v1.DnfStatusFlag_DNF_V5_CACHE_FOUND:
			hasCache = true
		}
	}
	if !hasRepo && !hasCache {
		return v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED
	}
	if hasRepo && hasCache {
		return v1.DnfMetadataStatus_AVAILABLE
	}
	return v1.DnfMetadataStatus_UNAVAILABLE
}

// discoverDnf4CachePresent checks for a DNF4 package cache at cacheDirPath.
func discoverDnf4CachePresent(hostPath, cacheDirPath string) (bool, error) {
	if cacheDirPath == "" {
		return false, nil
	}
	_, found, err := discoverDnf4CacheRepoDirPresence(hostPath, cacheDirPath)
	return found, err
}

// discoverDnf5CachePresent checks for a DNF5/libdnf5 package cache at cacheDirPath.
func discoverDnf5CachePresent(hostPath, cacheDirPath string) (bool, error) {
	if cacheDirPath == "" {
		return false, nil
	}
	_, found, err := discoverDnf5CacheRepoDirPresence(hostPath, cacheDirPath)
	return found, err
}

// discoverDnfStatusFlags probes the host filesystem for individual DNF-related
// facts and returns all that apply. DNF4 and DNF5 paths are probed independently.
func discoverDnfStatusFlags(hostPath string, dnf4ReposDirs, dnf5ReposDirs []string, dnf4CacheDirPath, dnf5CacheDirPath string) ([]v1.DnfStatusFlag, error) {
	var flags []v1.DnfStatusFlag
	var errs []error

	_, v4RepoFound, v4RepoErr := discoverDnfRepoFilePresence(hostPath, dnf4ReposDirs)
	_, v5RepoFound, v5RepoErr := discoverDnfRepoFilePresence(hostPath, dnf5ReposDirs)
	if v4RepoFound || v5RepoFound {
		flags = append(flags, v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND)
	} else {
		appendNonNil(&errs, v4RepoErr, v5RepoErr)
	}

	v4Cache, v4CacheErr := discoverDnf4CachePresent(hostPath, dnf4CacheDirPath)
	v5Cache, v5CacheErr := discoverDnf5CachePresent(hostPath, dnf5CacheDirPath)
	if v4Cache {
		flags = append(flags, v1.DnfStatusFlag_DNF_V4_CACHE_FOUND)
	}
	if v5Cache {
		flags = append(flags, v1.DnfStatusFlag_DNF_V5_CACHE_FOUND)
	}
	if !v4Cache && !v5Cache {
		appendNonNil(&errs, v4CacheErr, v5CacheErr)
	}

	switch hostprobe.DetectDNFVersion(hostPath) {
	case hostprobe.DNFVersion5:
		flags = append(flags, v1.DnfStatusFlag_DNF_V5_HISTORY_DB_FOUND)
	case hostprobe.DNFVersion4:
		flags = append(flags, v1.DnfStatusFlag_DNF_V4_HISTORY_DB_FOUND)
	}

	return flags, errors.Join(errs...)
}

func appendNonNil(errs *[]error, ee ...error) {
	for _, e := range ee {
		if e != nil {
			*errs = append(*errs, e)
		}
	}
}
