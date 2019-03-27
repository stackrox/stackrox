package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// OptionsMap contains the search options for a Node
var OptionsMap = blevesearch.Walk(v1.SearchCategory_NODES, "node", (*storage.Node)(nil))
