package mappings

import (
	"github.com/stackrox/stackrox/central/activecomponent/index/internal"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	search "github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap defines the search option for active contexts in active components.
var OptionsMap = search.Walk(v1.SearchCategory_ACTIVE_COMPONENT, "active_component", (*internal.IndexedContexts)(nil))
