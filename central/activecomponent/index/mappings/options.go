package mappings

import (
	"github.com/stackrox/rox/central/activecomponent/index/internal"
	v1 "github.com/stackrox/rox/generated/api/v1"
	search "github.com/stackrox/rox/pkg/search"
)

// OptionsMap defines the search option for active contexts in active components.
var OptionsMap = search.Walk(v1.SearchCategory_ACTIVE_COMPONENT, "active_component", (*internal.IndexedContexts)(nil))
