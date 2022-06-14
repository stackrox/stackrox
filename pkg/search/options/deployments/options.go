package deployments

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/processindicators"
)

var (
	imageMap = search.Walk(v1.SearchCategory_IMAGES, "image", (*storage.Image)(nil))

	// OptionsMap describes the options for Deployments
	OptionsMap = search.Walk(v1.SearchCategory_DEPLOYMENTS, "deployment", (*storage.Deployment)(nil)).
			Merge(processindicators.OptionsMap).
			Merge(imageMap)
)
