package discovery

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// discoverDnfCachePresent reports whether hostPath/cacheDirPath contains a
// subdirectory matching isRepoCacheDir. An empty cacheDirPath means "not
// probed for this DNF version" and reports false with no error.
func discoverDnfCachePresent(hostPath, cacheDirPath string, isRepoCacheDir func(os.DirEntry) bool) (bool, error) {
	if cacheDirPath == "" {
		return false, nil
	}
	cachePath := hostprobe.HostPathFor(hostPath, cacheDirPath)
	cacheEntries, err := os.ReadDir(cachePath)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", cachePath, err)
	}
	return slices.ContainsFunc(cacheEntries, isRepoCacheDir), nil
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

// discoverDnf4CachePresent reports whether a DNF4 package cache exists at
// cacheDirPath: a subdirectory whose name contains "-rpms-" under the cache
// root (e.g. /var/cache/dnf).
//
// DNF4 default layout (libdnf Const.hpp):
// https://github.com/rpm-software-management/libdnf/blob/53839f5bd88f378e57a1f1671b3db48d29984e24/libdnf/conf/Const.hpp
func discoverDnf4CachePresent(hostPath, cacheDirPath string) (bool, error) {
	return discoverDnfCachePresent(hostPath, cacheDirPath, func(e os.DirEntry) bool {
		return e.IsDir() && strings.Contains(e.Name(), "-rpms-")
	})
}

// discoverDnf5CachePresent reports whether a DNF5/libdnf5 package cache exists
// at cacheDirPath (e.g. /var/cache/libdnf5): any subdirectory indicates a
// repository cache layout.
//
// DNF5 defaults (libdnf5 const.hpp):
// https://github.com/rpm-software-management/dnf5/blob/185eaef1e0ad663bdb827a2179ab1df574a27d88/include/libdnf5/conf/const.hpp
func discoverDnf5CachePresent(hostPath, cacheDirPath string) (bool, error) {
	return discoverDnfCachePresent(hostPath, cacheDirPath, func(e os.DirEntry) bool {
		return e.IsDir()
	})
}

// discoverDnfStatusFlags probes the host filesystem for individual DNF-related
// facts and returns all that apply. DNF4 and DNF5 paths are probed independently.
func discoverDnfStatusFlags(hostPath string, dnf4ReposDirs, dnf5ReposDirs []string, dnf4CacheDirPath, dnf5CacheDirPath string) ([]v1.DnfStatusFlag, error) {
	var flags []v1.DnfStatusFlag
	var errs []error

	v4RepoFound, v4RepoErr := hostprobe.HasAnyRepoFileAt(hostPath, dnf4ReposDirs)
	v5RepoFound, v5RepoErr := hostprobe.HasAnyRepoFileAt(hostPath, dnf5ReposDirs)
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
