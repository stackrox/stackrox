package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap defines the search options for NetworkEntities.
var OptionsMap = search.Walk(v1.SearchCategory_NETWORK_ENTITY, "network_entity", (*storage.NetworkEntity)(nil))
