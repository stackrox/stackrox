package dackbox

import (
	"bytes"
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	sac "github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	imageCVESAC   = sac.ForResource(resources.Image)
	nodeCVESAC    = sac.ForResource(resources.Node)
	clusterCVESAC = sac.ForResource(resources.Cluster)
)

func existenceCheck(keyStr string) func(context.Context, []byte) bool {
	return func(ctx context.Context, key []byte) bool {
		return bytes.Contains(key, []byte(keyStr))
	}
}

func panicTransform(context.Context, []byte) [][]sac.ScopeKey {
	panic("unexpected called to transform")
}

func identity(readResources ...string) func(mockCtrl *gomock.Controller, t *testing.T) context.Context {
	return func(mockCtrl *gomock.Controller, t *testing.T) context.Context {
		identity := mocks.NewMockIdentity(mockCtrl)

		resourceToAccess := make(map[string]storage.Access)
		for _, resource := range readResources {
			resourceToAccess[resource] = storage.Access_READ_ACCESS
		}

		identity.EXPECT().Permissions().AnyTimes().Return(&storage.Role{
			ResourceToAccess: resourceToAccess,
		})
		return authn.ContextWithIdentity(context.Background(), identity, t)
	}
}

func TestNoSACApplyWithoutIdentity(t *testing.T) {
	filter, err := NewSharedObjectSACFilter(
		WithNode(nodeCVESAC, panicTransform, existenceCheck("node")),
		WithImage(imageCVESAC, panicTransform, existenceCheck("image")),
		WithCluster(clusterCVESAC, panicTransform, existenceCheck("cluster")),
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

			filtered, err := filter.Apply(c.ctx, c.keys...)
			require.NoError(t, err)
			assert.Equal(t, c.expectedKeys, filtered)
		})
	}
}

func TestNoSACApplyWithIdentity(t *testing.T) {
	filter, err := NewSharedObjectSACFilter(
		WithNode(nodeCVESAC, panicTransform, existenceCheck("node")),
		WithImage(imageCVESAC, panicTransform, existenceCheck("image")),
		WithCluster(clusterCVESAC, panicTransform, existenceCheck("cluster")),
		WithSharedObjectAccess(storage.Access_READ_ACCESS))
	require.NoError(t, err)

	cases := []struct {
		name               string
		identityFunc       func(mockCtrl *gomock.Controller, t *testing.T) context.Context
		keys, expectedKeys []string
	}{
		{
			name:         "all access",
			identityFunc: identity("Node", "Image", "Cluster"),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"node", "image", "cluster", "node-image", "image-cluster"},
		},
		{
			name:         "no access",
			identityFunc: identity(),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: nil,
		},
		{
			name:         "node access",
			identityFunc: identity("Node"),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"node", "node-image"},
		},
		{
			name:         "image access",
			identityFunc: identity("Image"),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"image", "node-image", "image-cluster"},
		},
		{
			name:         "cluster access",
			identityFunc: identity("Cluster"),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"cluster", "image-cluster"},
		},
		{
			name:         "image and node access",
			identityFunc: identity("Image", "Node"),
			keys:         []string{"node", "image", "cluster", "node-image", "image-cluster"},
			expectedKeys: []string{"node", "image", "node-image", "image-cluster"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx := c.identityFunc(mockCtrl, t)
			filtered, err := filter.Apply(ctx, c.keys...)
			require.NoError(t, err)
			assert.Equal(t, c.expectedKeys, filtered)
		})
	}
}
