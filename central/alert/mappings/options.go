package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
)

// TODO: [ROX-10206] Reconcile storage.ListAlert search terms with storage.Alert

// OptionsMap is exposed for e2e test.
var OptionsMap search.OptionsMap

func init() {
	OptionsMap = search.Walk(v1.SearchCategory_ALERTS, "list_alert", (*storage.ListAlert)(nil))
	alertOptions := schema.AlertsSchema.OptionsMap

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
