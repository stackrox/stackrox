package images

import (
	"github.com/stackrox/rox/central/views/common"
)

type imageResponse struct {
	common.ResourceCountByImageCVESeverity
	ImageID string `db:"image_sha"`
}

func (i *imageResponse) GetImageID() string {
	return i.ImageID
}

func (i *imageResponse) GetImageCVEsBySeverity() common.ResourceCountByCVESeverity {
	return &i.ResourceCountByImageCVESeverity
}
