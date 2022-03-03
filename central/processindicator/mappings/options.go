package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test.
var OptionsMap = search.Walk(v1.SearchCategory_PROCESS_INDICATORS, "process_indicator", (*storage.ProcessIndicator)(nil))
