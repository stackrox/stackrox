package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/views/platformcve"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("ClusterCountByType", []string{
			"generic: Int!",
			"kubernetes: Int!",
			"openshift: Int!",
			"openshift4: Int!",
		}),
	)
}

type clusterCountByTypeResolver struct {
	ctx  context.Context
	root *Resolver
	data platformcve.ClusterCountByPlatformType
}

func (resolver *Resolver) wrapClusterCountByTypeWithContext(ctx context.Context, value platformcve.ClusterCountByPlatformType, err error) (*clusterCountByTypeResolver, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &clusterCountByTypeResolver{ctx: ctx, root: resolver, data: platformcve.NewEmptyClusterCountByPlatformType()}, nil
	}
	return &clusterCountByTypeResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Generic returns number of clusters of type GENERIC
func (resolver *clusterCountByTypeResolver) Generic(_ context.Context) int32 {
	return int32(resolver.data.GetGenericClusterCount())
}

// Kubernetes returns the number of clusters of type KUBERNETES
func (resolver *clusterCountByTypeResolver) Kubernetes(_ context.Context) int32 {
	return int32(resolver.data.GetKubernetesClusterCount())
}

// Openshift returns the number of clusters of type OPENSHIFT
func (resolver *clusterCountByTypeResolver) Openshift(_ context.Context) int32 {
	return int32(resolver.data.GetOpenshiftClusterCount())
}

// Openshift4 retruns the number of clusters of type OPENSHIFT4
func (resolver *clusterCountByTypeResolver) Openshift4(_ context.Context) int32 {
	return int32(resolver.data.GetOpenshift4ClusterCount())
}
