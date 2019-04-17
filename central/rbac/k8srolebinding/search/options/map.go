package options

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Map is the map of indexed fields in k8s rolebindings objects.
var Map = blevesearch.Walk(v1.SearchCategory_ROLEBINDINGS, "k8srolebinding", (*storage.K8SRoleBinding)(nil))
