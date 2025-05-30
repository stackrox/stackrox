package imagecomponentflat

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// ComponentFlat is an interface to get image component properties.
//
//go:generate mockgen-wrapper
type ComponentFlat interface {
	GetComponent() string
	GetComponentIDs() []string
	GetVersion() string
	GetTopCVSS() float32
	GetRiskScore() float32
	GetOperatingSystem() string
}

// ComponentFlatView interface is like a SQL view that provides functionality to fetch the image component data
// irrespective of the data model.
//
//go:generate mockgen-wrapper
type ComponentFlatView interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Get(ctx context.Context, q *v1.Query) ([]ComponentFlat, error)
}
