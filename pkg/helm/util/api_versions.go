package util

import (
	"strings"
	"unicode"

	"github.com/stackrox/stackrox/pkg/set"
	"helm.sh/helm/v3/pkg/chartutil"
)

// VersionSetFromResources creates a set of API versions from the given list of resources.
// I.e., if the list contains `apps/v1/Deployment`, the resulting set will contain
// `apps/v1/Deployment` as well as `apps/v1`.
func VersionSetFromResources(resources ...string) chartutil.VersionSet {
	allVersions := set.NewStringSet(resources...)
	for _, resource := range resources {
		lastSlashIdx := strings.LastIndex(resource, "/")
		if lastSlashIdx == -1 || lastSlashIdx == len(resource)-1 {
			continue
		}
		if !unicode.IsUpper(rune(resource[lastSlashIdx+1])) {
			continue
		}
		allVersions.Add(resource[:lastSlashIdx])
	}

	return allVersions.AsSlice()
}
