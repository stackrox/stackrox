package options

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is the mapping for the test struct
var OptionsMap = search.Walk(v1.SearchCategory_SEARCH_UNSET, "single", (*storage.TestSingleKeyStruct)(nil))
