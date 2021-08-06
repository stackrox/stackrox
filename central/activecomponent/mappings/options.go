package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap defines the search options for active components stored in edges.
var OptionsMap = search.Walk(v1.SearchCategory_ACTIVE_COMPONENT, "active_component", (*storage.ActiveComponent)(nil))
