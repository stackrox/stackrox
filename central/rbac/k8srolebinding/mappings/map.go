package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is the map of indexed fields in k8s rolebindings objects.
var OptionsMap = search.Walk(v1.SearchCategory_ROLEBINDINGS, "k8s_role_binding", (*storage.K8SRoleBinding)(nil))
