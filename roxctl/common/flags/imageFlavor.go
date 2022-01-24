package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
)

var (
	imageFlavorDefault string = defaults.ImageFlavorNameRHACSRelease
)

func init() {
	if !buildinfo.ReleaseBuild {
		imageFlavorDefault = defaults.ImageFlavorNameDevelopmentBuild
	}
}

// AddImageDefaults adds the image-defaults flag to the command.
func AddImageDefaults(pf *pflag.FlagSet, destination *string) {
	imageFlavorHelpStr := fmt.Sprintf("default container images settings (%v); it controls repositories from where to download the images, image names and tags format",
		strings.Join(defaults.GetAllowedImageFlavorNames(buildinfo.ReleaseBuild), ", "))
	pf.StringVar(destination, "image-defaults", imageFlavorDefault, imageFlavorHelpStr)
}
