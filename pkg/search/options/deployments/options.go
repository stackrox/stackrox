package deployments

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/processindicators"
)

// OptionsMap describes the options for Deployments
var OptionsMap = search.Walk(v1.SearchCategory_DEPLOYMENTS, "deployment", (*storage.Deployment)(nil)).
	Merge(processindicators.OptionsMap)
