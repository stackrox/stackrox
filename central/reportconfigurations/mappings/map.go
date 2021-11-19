package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is the map of indexed fields in storage.ReportConfig object.
var OptionsMap = search.Walk(v1.SearchCategory_REPORT_CONFIGURATIONS, "report_configuration", (*storage.ReportConfiguration)(nil))
