package dackbox

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ImageParts represents the pieces of data in an image.
type ImageParts struct {
	image     *storage.Image
	listImage *storage.ListImage

	children []ComponentParts
}

// ComponentParts represents the pieces of data in an image component.
type ComponentParts struct {
	edge      *storage.ImageComponentEdge
	component *storage.ImageComponent

	children []CVEParts
}

// CVEParts represents the pieces of data in a CVE.
type CVEParts struct {
	edge *storage.ComponentCVEEdge
	cve  *storage.CVE
}
