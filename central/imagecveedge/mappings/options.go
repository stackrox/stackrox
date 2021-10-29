package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap defines the search options for vulnerabilities in images.
var OptionsMap = search.Walk(v1.SearchCategory_IMAGE_VULN_EDGE, "image_c_v_e_edge", (*storage.ImageCVEEdge)(nil))
