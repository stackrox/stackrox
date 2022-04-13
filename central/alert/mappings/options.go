package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap is exposed for e2e test.
var OptionsMap = search.Walk(v1.SearchCategory_ALERTS, "list_alert", (*storage.ListAlert)(nil))
