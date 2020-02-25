package sac

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	namespaceDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	imageComponentSAC = sac.ForResource(resources.ImageComponent)

	imageComponentClusterPath = [][]byte{
		componentDackBox.Bucket,
		imageDackBox.Bucket,
		deploymentDackBox.Bucket,
		namespaceDackBox.Bucket,
		clusterDackBox.Bucket,
	}

	imageComponentNamespacePath = [][]byte{
		componentDackBox.Bucket,
		imageDackBox.Bucket,
		deploymentDackBox.Bucket,
		namespaceDackBox.Bucket,
	}

	imageComponentSACFilter filtered.Filter
	once                    sync.Once
)

// GetSACFilter returns the sac filter for image component ids.
func GetSACFilter(graphProvider graph.Provider) filtered.Filter {
	once.Do(func() {
		var err error
		imageComponentSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(imageComponentSAC),
			filtered.WithGraphProvider(graphProvider),
			filtered.WithClusterPath(imageComponentClusterPath...),
			filtered.WithNamespacePath(imageComponentNamespacePath...),
		)
		utils.Must(err)
	})
	return imageComponentSACFilter
}
