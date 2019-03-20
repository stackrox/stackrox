package kubernetes

import (
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// GetResolvedRegistry returns the registry endpoint from the image definition
func GetResolvedRegistry(image string) (string, error) {
	parsedImage := utils.GenerateImageFromStringIgnoringError(image)
	return urlfmt.FormatURL(parsedImage.GetName().GetRegistry(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
}
