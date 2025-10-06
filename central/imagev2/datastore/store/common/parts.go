package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ImagePartsV2 represents the pieces of data in an image.
type ImagePartsV2 struct {
	Image    *storage.ImageV2
	Children []ComponentPartsV2
}

// ComponentPartsV2 represents the pieces of data in an image component.
type ComponentPartsV2 struct {
	ComponentV2 *storage.ImageComponentV2
	Children    []CVEPartsV2
}

// CVEPartsV2 represents the pieces of data in a CVE.
type CVEPartsV2 struct {
	CVEV2 *storage.ImageCVEV2
}
