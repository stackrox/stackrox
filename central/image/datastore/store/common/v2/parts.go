package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ImageParts represents the pieces of data in an image.
type ImageParts struct {
	Image *storage.Image

	Children []ComponentParts
	// imageCVEEdges stores CVE ID to *storage.imageCVEEdge object mappings
	ImageCVEEdges map[string]*storage.ImageCVEEdge
}

// ComponentParts represents the pieces of data in an image component.
type ComponentParts struct {
	Edge        *storage.ImageComponentEdge
	Component   *storage.ImageComponent
	ComponentV2 *storage.ImageComponentV2

	Children []CVEParts
}

// CVEParts represents the pieces of data in a CVE.
type CVEParts struct {
	Edge  *storage.ComponentCVEEdge
	CVE   *storage.ImageCVE
	CVEV2 *storage.ImageCVEV2
}
