package imagecve

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// ImageCVECore is an interface to get image CVE properties.
type ImageCVECore interface {
	GetCVE() string
	GetTopCVSS() float32
	GetAffectedImages() int
}

// ImageCVEView interface is like a SQL view that provides functionality to fetch the image CVE data
// irrespective of the data model. One CVE can have multiple database entries if that CVE impacts multiple distros.
// Each record may have different values for properties like severity. However, the core information is the same.
// Core information such as universal CVE identifier, summary, etc. is constant.
type ImageCVEView interface {
	Get(ctx context.Context, q *v1.Query) ([]ImageCVECore, error)
}
