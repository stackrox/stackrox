package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// TODO: [ROX-10206] Reconcile storage.ListAlert search terms with storage.Alert

// OptionsMap is exposed for e2e test.
var OptionsMap = search.Walk(v1.SearchCategory_ALERTS, "list_alert", (*storage.ListAlert)(nil))
