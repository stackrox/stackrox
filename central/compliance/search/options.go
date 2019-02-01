package search

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test
var OptionsMap = search.OptionsMapFromMap(map[search.FieldLabel]*v1.SearchField{
	search.Cluster:  nil,
	search.Standard: nil,
})
