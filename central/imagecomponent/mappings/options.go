package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap defines the search options for image components stored in images.
var OptionsMap = search.Walk(v1.SearchCategory_IMAGE_COMPONENTS, "image_component", (*storage.ImageComponent)(nil))
