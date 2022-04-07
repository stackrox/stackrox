package dackbox

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	imageCVESAC   = sac.ForResource(resources.Image)
	nodeCVESAC    = sac.ForResource(resources.Node)
	clusterCVESAC = sac.ForResource(resources.Cluster)
)

var (
	testGraph = testutils.GraphFromPaths(
		dackbox.BackwardsPath(
			cveDackBox.BucketHandler.GetKey("node"),
			componentDackBox.BucketHandler.GetKey("component1"),
			nodeDackBox.BucketHandler.GetKey("node1"),
			clusterDackBox.BucketHandler.GetKey("cluster1"),
		),
		dackbox.BackwardsPath(
			cveDackBox.BucketHandler.GetKey("image"),
			componentDackBox.BucketHandler.GetKey("component2"),
			imageDackBox.BucketHandler.GetKey("image1"),
			deploymentDackBox.BucketHandler.GetKey("deploy1"),
			nsDackBox.BucketHandler.GetKey("ns1"),
			clusterDackBox.BucketHandler.GetKey("cluster2"),
		),
		dackbox.BackwardsPath(
			cveDackBox.BucketHandler.GetKey("cluster"),
			clusterDackBox.BucketHandler.GetKey("cluster3"),
		),
		dackbox.BackwardsPath(
			cveDackBox.BucketHandler.GetKey("node-image"),
			componentDackBox.BucketHandler.GetKey("component3"),
			nodeDackBox.BucketHandler.GetKey("node1"),
		),
		dackbox.BackwardsPath(
			cveDackBox.BucketHandler.GetKey("node-image"),
			componentDackBox.BucketHandler.GetKey("component4"),
			imageDackBox.BucketHandler.GetKey("image1"),
		),
		dackbox.BackwardsPath(
			cveDackBox.BucketHandler.GetKey("image-cluster"),
			componentDackBox.BucketHandler.GetKey("component5"),
			imageDackBox.BucketHandler.GetKey("image2"),
			deploymentDackBox.BucketHandler.GetKey("deploy1"),
		),
		dackbox.BackwardsPath(
			cveDackBox.BucketHandler.GetKey("image-cluster"),
			clusterDackBox.BucketHandler.GetKey("cluster2"),
		),
	)
)

func TestNoSACApplyWithoutIdentity(t *testing.T) {
	filter, err := NewSharedObjectSACFilter(
		WithNode(nodeCVESAC, CVEToNodeBucketPath),
		WithImage(imageCVESAC, CVEToImageBucketPath),
		WithCluster(clusterCVESAC, CVEToClusterBucketPath),
		WithSharedObjectAccess(storage.Access_READ_ACCESS))
	require.NoError(t, err)

	cases := []struct {
		name               string
		ctx                context.Context
		keys, expectedKeys []string
	}{
		{
			name:         "all access",
			ctx:          sac.WithAllAccess(context.Background()),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"node", "image", "cluster", "node-image", "image-cluster"},
		},
		{
			name:         "no access",
			ctx:          sac.WithNoAccess(context.Background()),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: nil,
		},
		{
			name: "node access",
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Node))),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"node", "node-image"},
		},
		{
			name: "image access",
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Image))),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"image", "node-image", "image-cluster"},
		},
		{
			name: "cluster access",
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster))),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"cluster", "image-cluster"},
		},
		{
			name: "image and node access",
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Image, resources.Node))),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"node", "image", "node-image", "image-cluster"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			testutils.DoWithGraph(c.ctx, testGraph, func(ctx context.Context) {
				filteredIndices, all, err := filter.Apply(ctx, c.keys...)
				require.NoError(t, err)
				if all {
					assert.Equal(t, c.expectedKeys, c.keys)
				} else {
					assert.Equal(t, c.expectedKeys, sliceutils.StringSelect(c.keys, filteredIndices...))
				}
			})
		})
	}
}
