package options

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Map is the map of indexed fields in k8s role objects
var Map = blevesearch.Walk(v1.SearchCategory_ROLES, "k8srole", (*storage.K8SRole)(nil))
