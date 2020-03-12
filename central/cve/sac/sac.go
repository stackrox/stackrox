package sac

import (
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
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
	cveSAC = sac.ForResource(resources.CVE)

	clusterCVEClusterPath = [][]byte{
		cveDackBox.Bucket,
		clusterDackBox.Bucket,
	}

	cveClusterPath = [][]byte{
		cveDackBox.Bucket,
		componentDackBox.Bucket,
		imageDackBox.Bucket,
		deploymentDackBox.Bucket,
		namespaceDackBox.SACBucket,
		clusterDackBox.Bucket,
	}

	cveNamespacePath = [][]byte{
		cveDackBox.Bucket,
		componentDackBox.Bucket,
		imageDackBox.Bucket,
		deploymentDackBox.Bucket,
		namespaceDackBox.SACBucket,
	}

	clusterCVESACFilter filtered.Filter
	cveSACFilter        filtered.Filter
	once                sync.Once
)

// GetSACFilters returns the sac filter for cve ids.
func GetSACFilters(graphProvider graph.Provider) []filtered.Filter {
	once.Do(func() {
		var err error
		cveSACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(cveSAC),
			filtered.WithGraphProvider(graphProvider),
			filtered.WithClusterPath(cveClusterPath...),
			filtered.WithNamespacePath(cveNamespacePath...),
		)
		utils.Must(err)

		clusterCVESACFilter, err = filtered.NewSACFilter(
			filtered.WithResourceHelper(cveSAC),
			filtered.WithGraphProvider(graphProvider),
			filtered.WithClusterPath(clusterCVEClusterPath...),
		)
		utils.Must(err)
	})
	return []filtered.Filter{cveSACFilter, clusterCVESACFilter}
}
