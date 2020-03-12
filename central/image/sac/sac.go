package sac

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	namespaceDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	imageSAC = sac.ForResource(resources.Image)

	imageClusterPath = [][]byte{
		imageDackBox.Bucket,
		deploymentDackBox.Bucket,
		namespaceDackBox.SACBucket,
		clusterDackBox.Bucket,
	}

	imageNamespacePath = [][]byte{
		imageDackBox.Bucket,
		deploymentDackBox.Bucket,
		namespaceDackBox.SACBucket,
	}

	imageSACFilter filtered.Filter
	once           sync.Once
)

// GetSACFilter returns the sac filter for image ids.
func GetSACFilter(graphProvider graph.Provider) filtered.Filter {
	once.Do(func() {
		var err error
		imageSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageSAC),
			filtered.WithGraphProvider(graphProvider),
			filtered.WithClusterPath(imageClusterPath...),
			filtered.WithNamespacePath(imageNamespacePath...),
		)
		utils.Must(err)
	})
	return imageSACFilter
}
