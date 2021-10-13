package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test.
var OptionsMap search.OptionsMap

func init() {
	if features.PostgresPOC.Enabled() {
		OptionsMap = search.Walk(v1.SearchCategory_ALERTS, "alert", (*storage.Alert)(nil))
	} else {
		OptionsMap = search.Walk(v1.SearchCategory_ALERTS, "list_alert", (*storage.ListAlert)(nil))
	}
}
