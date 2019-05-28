package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// OptionsMap is the map of indexed fields in k8s role objects
var OptionsMap = blevesearch.Walk(v1.SearchCategory_ROLES, "k8s_role", (*storage.K8SRole)(nil))
