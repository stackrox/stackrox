package processindicators

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	storage "github.com/stackrox/stackrox/generated/storage"
	search "github.com/stackrox/stackrox/pkg/search"
)

var (
	// ProcessPrefix defines the prefix for search when using process indicators. This is exported so that we can properly
	// alias across indexes when search indicators through deployments
	ProcessPrefix = "process_indicator"

	// OptionsMap defines the options for process indicators
	OptionsMap = search.Walk(v1.SearchCategory_PROCESS_INDICATORS, ProcessPrefix, (*storage.ProcessIndicator)(nil))
)
