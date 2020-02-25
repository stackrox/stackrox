package sac

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	namespaceDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	nsSAC = sac.ForResource(resources.Namespace)

	namespaceClusterPath = [][]byte{
		namespaceDackBox.Bucket,
		clusterDackBox.Bucket,
	}

	nsSACFilter filtered.Filter
	once        sync.Once
)

// GetSACFilter returns the sac filter for image ids.
func GetSACFilter(graphProvider graph.Provider) filtered.Filter {
	once.Do(func() {
		var err error
		nsSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(nsSAC),
			filtered.WithGraphProvider(graphProvider),
			filtered.WithClusterPath(namespaceClusterPath...),
			filtered.WithNamespacePath(namespaceDackBox.Bucket),
		)
		utils.Must(err)
	})
	return nsSACFilter
}
