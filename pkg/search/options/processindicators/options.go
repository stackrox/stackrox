package processindicators

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
)

// OptionsMap defines the options for process indicators
var OptionsMap = search.Walk(v1.SearchCategory_PROCESS_INDICATORS, "process_indicator", (*storage.ProcessIndicator)(nil))
