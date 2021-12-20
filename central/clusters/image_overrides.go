package clusters

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/images/utils"
)

type imageOverrides struct {
	flavor         *defaults.ImageFlavor
	mainImage      *storage.ImageName
	collectorImage *storage.ImageName
}

// NewImageOverrides creates an instance of imageOverrides; responsible for determining collector full and slim images
// values.
func NewImageOverrides(flavor *defaults.ImageFlavor, c *storage.Cluster) (*imageOverrides, error) {
	var mainImageName, collectorImageName *storage.ImageName
	mainImage, err := utils.GenerateImageFromStringWithDefaultTag(c.MainImage, flavor.MainImageTag)
	if err != nil {
		return nil, err
	}
	mainImageName = mainImage.GetName()

	if c.CollectorImage != "" {
		collectorImage, err := utils.GenerateImageFromString(c.CollectorImage)
		if err != nil {
			return nil, err
		}
		collectorImageName = collectorImage.GetName()
	}

	return &imageOverrides{
		flavor:         flavor,
		mainImage:      mainImageName,
		collectorImage: collectorImageName,
	}, nil
}

func (img *imageOverrides) isMainImageDefault() bool {
	overrideImageNoTag := fmt.Sprintf("%s/%s", img.mainImage.Registry, img.mainImage.Remote)
	return img.flavor.MainImageNoTag() == overrideImageNoTag
}

// SetMainOverride adds main image values to meta values as defined in secured cluster object.
func (img *imageOverrides) SetMainOverride(metaValues charts.MetaValues) {
	metaValues["MainRegistry"] = img.mainImage.Registry
	metaValues["ImageRemote"] = img.mainImage.Remote
	metaValues["ImageTag"] = img.mainImage.Tag
}

// SetCollectorFullOverride adds collector full image reference to meta values object. The collector repository defined
// in the cluster object can be passed from roxctl or as direct input in the UI when creating a new secured cluster.
// If no value is passed, the collector image will be derived from the main image. For example:
// main image: "quay.io/rhacs/main" => collector image: "quay.io/rhacs/collector"
func (img *imageOverrides) SetCollectorFullOverride(metaValues charts.MetaValues) {
	if img.collectorImage != nil {
		metaValues["CollectorRegistry"] = img.collectorImage.Registry
		metaValues["CollectorFullImageRemote"] = img.collectorImage.Remote
	} else {
		if img.isMainImageDefault() {
			metaValues["CollectorRegistry"] = img.flavor.CollectorRegistry
			metaValues["CollectorFullImageRemote"] = img.flavor.CollectorImageName
		} else {
			derivedImage := defaultimages.GenerateNamedImageFromMainImage(img.mainImage, img.flavor.CollectorImageTag,
				img.flavor.CollectorImageName)
			metaValues["CollectorRegistry"] = derivedImage.Registry
			metaValues["CollectorFullImageRemote"] = derivedImage.Remote
		}
	}
	metaValues["CollectorFullImageTag"] = img.flavor.CollectorImageTag
}

// SetCollectorSlimOverride adds collector slim image reference to meta values object. Slim collector will be derived
// similarly to SetCollectorFullOverride. However, if a collector registry is specified and current flavor has different
// image names for collector slim and full: collector slim has to be derived from full instead. For example:
// collector full image: "custom.registry.io/collector" => collector slim image: "custom.registry.io/collector-slim"
func (img *imageOverrides) SetCollectorSlimOverride(metaValues charts.MetaValues) {
	if img.collectorImage != nil {
		derivedImage := defaultimages.GenerateNamedImageFromMainImage(img.collectorImage, img.flavor.CollectorSlimImageTag,
			img.flavor.CollectorSlimImageName)
		metaValues["CollectorSlimImageRemote"] = derivedImage.Remote
	} else {
		if img.isMainImageDefault() {
			metaValues["CollectorSlimImageRemote"] = img.flavor.CollectorSlimImageName
		} else {
			derivedImage := defaultimages.GenerateNamedImageFromMainImage(img.mainImage, img.flavor.CollectorSlimImageTag,
				img.flavor.CollectorSlimImageName)
			metaValues["CollectorSlimImageRemote"] = derivedImage.Remote
		}
	}
	metaValues["CollectorSlimImageTag"] = img.flavor.CollectorSlimImageTag
}
