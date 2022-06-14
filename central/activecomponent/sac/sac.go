package sac

import (
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
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
