package mappings

import (
	"github.com/stackrox/rox/central/deployment/index/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// OptionsMap is exposed for e2e test
var OptionsMap = blevesearch.Walk(v1.SearchCategory_IMAGES, "image", (*storage.Image)(nil)).
	Add(search.Cluster, mappings.OptionsMap.MustGet(search.Cluster.String())).
	Add(search.Namespace, mappings.OptionsMap.MustGet(search.Namespace.String())).
	Add(search.Label, mappings.OptionsMap.MustGet(search.Label.String()))
