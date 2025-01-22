package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
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
	alertOptions := schema.AlertsSchema.OptionsMap.Clone()

	// There are more search terms in the alert proto due to the embeddings of policies.
	// This pruning of options ensures that the search options are stable between RocksDB and Postgres
	// while also ensuring that highlights work
	for opt := range alertOptions.Original() {
		if _, ok := OptionsMap.Get(string(opt)); !ok {
			alertOptions.Remove(opt)
		}
	}
	OptionsMap = alertOptions

}
