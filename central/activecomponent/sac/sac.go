package sac

import (
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	deploymentSAC = sac.ForResource(resources.Deployment)

	activeComponentSACFilter filtered.Filter
	once                     sync.Once
)

// GetSACFilter returns the sac filter for active component ids.
func GetSACFilter() filtered.Filter {
	once.Do(func() {
		var err error
		activeComponentSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(deploymentSAC),
			filtered.WithScopeTransform(dackbox.ActiveComponentSACTransform),
			filtered.WithReadAccess(),
		)
		utils.CrashOnError(err)
	})
	return activeComponentSACFilter
}
