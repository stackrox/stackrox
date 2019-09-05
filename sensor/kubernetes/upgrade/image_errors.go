package upgrade

import "github.com/stackrox/rox/pkg/set"

var (
	imagePullRelatedReasons = set.NewFrozenStringSet(
		"ImagePullBackOff",
		"ImageInspectError",
		"ErrImagePull",
		"ErrImageNeverPull",
		"RegistryUnavailable",
		"InvalidImageName",
	)
)

func isImagePullRelatedReason(reason string) bool {
	return imagePullRelatedReasons.Contains(reason)
}
