package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap is the map of indexed fields in k8s role objects
var OptionsMap = search.Walk(v1.SearchCategory_ROLES, "k8s_role", (*storage.K8SRole)(nil))
