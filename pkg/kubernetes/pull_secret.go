package kubernetes

import (
	"github.com/stackrox/stackrox/pkg/images/utils"
	"github.com/stackrox/stackrox/pkg/urlfmt"
)

// GetResolvedRegistry returns the registry endpoint from the image definition
func GetResolvedRegistry(image string) (string, error) {
	parsedImage, err := utils.GenerateImageFromString(image)
	if err != nil {
		return "", err
	}
	return urlfmt.FormatURL(parsedImage.GetName().GetRegistry(), urlfmt.HTTPS, urlfmt.NoTrailingSlash), nil
}
