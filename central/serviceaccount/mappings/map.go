package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap is the map of indexed fields in service account and relationship objects.
var OptionsMap = search.Walk(v1.SearchCategory_SERVICE_ACCOUNTS, "service_account", (*storage.ServiceAccount)(nil))
