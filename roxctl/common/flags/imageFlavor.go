package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
)

const (
	// FlagNameImageDefaults is a shared constant for --image-defaults command line flag.
	FlagNameImageDefaults = "image-defaults"
	// FlagNameMainImage is a shared constant for --main-image command line flag.
	FlagNameMainImage = "main-image"
	// FlagNameCentralDBImage is a shared constant for --central-db-image command line flag.
	FlagNameCentralDBImage = "central-db-image"
	// FlagNameScannerImage is a shared constant for --scanner-image command line flag.
	FlagNameScannerImage = "scanner-image"
	// FlagNameScannerDBImage is a shared constant for --scanner-db-image command line flag.
	FlagNameScannerDBImage = "scanner-db-image"
)

var (
	imageFlavorDefault = defaults.ImageFlavorNameRHACSRelease
)

// ImageDefaultsFlagName is a shared constant for --image-defaults command line flag.
const ImageDefaultsFlagName = "image-defaults"

func init() {
	if !buildinfo.ReleaseBuild {
		imageFlavorDefault = defaults.ImageFlavorNameDevelopmentBuild
	}
}

// AddImageDefaults adds the image-defaults flag to the command.
func AddImageDefaults(pf *pflag.FlagSet, destination *string) {
	imageFlavorHelpStr := fmt.Sprintf("default container images settings (%v); it controls repositories from where to download the images, image names and tags format",
		strings.Join(defaults.GetVisibleImageFlavorNames(buildinfo.ReleaseBuild), ", "))
	pf.StringVar(destination, FlagNameImageDefaults, imageFlavorDefault, imageFlavorHelpStr)
}
