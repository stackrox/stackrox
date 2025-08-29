package images

import (
	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/pkg/features"
)

type imageResponse struct {
	common.ResourceCountByImageCVESeverity
	ImageID   string `db:"image_sha"`
	ImageV2ID string `db:"id"`
}

func (i *imageResponse) GetImageID() string {
	if features.FlattenImageData.Enabled() {
		return i.ImageV2ID
	}
	return i.ImageID
}

func (i *imageResponse) GetImageCVEsBySeverity() common.ResourceCountByCVESeverity {
	return &i.ResourceCountByImageCVESeverity
}
