package options

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap is the mapping for the test struct
var OptionsMap = search.Walk(v1.SearchCategory_SEARCH_UNSET, "multi", (*storage.TestMultiKeyStruct)(nil))
