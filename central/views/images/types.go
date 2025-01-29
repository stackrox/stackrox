package images

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// ImageCore is an interface to get image properties.
//
//go:generate mockgen-wrapper
type ImageCore interface {
	GetImageID() string
	GetImageCVEsBySeverity() common.ResourceCountByCVESeverity
}

// ImageView interface provides functionality to fetch the image data
type ImageView interface {
	Get(ctx context.Context, q *v1.Query) ([]ImageCore, error)
}
