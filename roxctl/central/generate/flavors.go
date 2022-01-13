package generate

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/images/defaults"
)

// GetValidImageDefaults returns valid possible values of the `--image-defaults` parameter
func GetValidImageDefaults(isRelease bool) []string {
	m := imageDefaultsMap(isRelease)
	result := make([]string, 0, len(m))
	for key := range m {
		if key != "" {
			result = append(result, key)
		}
	}
	return result
}

// imageDefaultsMap maps the value of roxctl's '--image-defaults' parameter to a (function returing) flavor
func imageDefaultsMap(isRelease bool) map[string]func() defaults.ImageFlavor {
	m := make(map[string]func() defaults.ImageFlavor)
	if isRelease {
		m[""] = defaults.StackRoxIOReleaseImageFlavor // TODO(RS-380): set default to RHACS
	} else {
		m[""] = defaults.DevelopmentBuildImageFlavor
		m["development"] = defaults.DevelopmentBuildImageFlavor
	}
	m["stackrox.io"] = defaults.StackRoxIOReleaseImageFlavor
	// m["rhacs"] = RHACSReleaseImageFlavor // TODO(RS-380): uncomment to enable rhacs flavor
	return m
}

// GetImageFlavorByRoxctlFlag returns flavor object based on the value of --image-defaults parameter in roxctl
func GetImageFlavorByRoxctlFlag(flag string, isRelease bool) (defaults.ImageFlavor, error) {
	if fn, ok := imageDefaultsMap(isRelease)[flag]; ok {
		return fn(), nil
	}
	return defaults.ImageFlavor{}, fmt.Errorf("invalid value of '--image-defaults=%s', allowed values: %s", flag, strings.Join(GetValidImageDefaults(isRelease), ", "))
}
