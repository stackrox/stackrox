package upgrade

import "github.com/stackrox/stackrox/pkg/set"

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
