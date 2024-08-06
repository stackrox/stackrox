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

func getImageErrorRemediation(reason string) string {
	switch reason {
	case "ImagePullBackOff":
		fallthrough
	case "ErrImagePull":
		return " This typically happens when the image does not exist or Sensor lacks required credentials to pull it."
	}
	return ""
}
