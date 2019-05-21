package mappings

import (
	processIndicatorMapping "github.com/stackrox/rox/central/processindicator/index/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// OptionsMap describes the options for Deployments
var OptionsMap = blevesearch.Walk(v1.SearchCategory_DEPLOYMENTS, "deployment", (*storage.Deployment)(nil)).
	Merge(processIndicatorMapping.OptionsMap)
