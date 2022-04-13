package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap is the map of indexed fields in k8s rolebindings objects.
var OptionsMap = search.Walk(v1.SearchCategory_ROLEBINDINGS, "k8s_role_binding", (*storage.K8SRoleBinding)(nil))
