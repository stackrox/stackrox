package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap is the map of indexed fields in storage.ReportConfig object.
var OptionsMap = search.Walk(v1.SearchCategory_REPORT_CONFIGURATIONS, "report_configuration", (*storage.ReportConfiguration)(nil))
