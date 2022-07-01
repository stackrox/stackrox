package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var OptionsMap = search.Walk(v1.SearchCategory_IMAGE_INTEGRATIONS, "image_integration", (*storage.ImageIntegration)(nil))
