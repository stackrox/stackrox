package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test.
var (
	OptionsMap      search.OptionsMap
	optionsToRemove = [...]search.FieldLabel{"SORT_Lifecycle Stage", "SORT_Enforcement"}
)

func init() {
	OptionsMap = search.Walk(v1.SearchCategory_ALERTS, "alert", (*storage.Alert)(nil))
	for _, opt := range optionsToRemove {
		OptionsMap.Remove(opt)
	}
}
