package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap defines the search options for vulnerabilities in images.
var OptionsMap = search.Walk(v1.SearchCategory_IMAGE_VULN_EDGE, "image_c_v_e_edge", (*storage.ImageCVEEdge)(nil))
