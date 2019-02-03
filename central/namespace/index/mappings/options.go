package mappings

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// OptionsMap defines the options for search by namespace
var OptionsMap = blevesearch.Walk(v1.SearchCategory_NAMESPACES, "namespace", (*storage.Namespace)(nil))
