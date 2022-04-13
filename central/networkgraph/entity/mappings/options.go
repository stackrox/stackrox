package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap defines the search options for NetworkEntities.
var OptionsMap = search.Walk(v1.SearchCategory_NETWORK_ENTITY, "network_entity", (*storage.NetworkEntity)(nil))
