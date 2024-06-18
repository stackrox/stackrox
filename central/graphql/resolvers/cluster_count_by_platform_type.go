package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/views/platformcve"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("ClusterCountByPlatformType", []string{
			"generic: Int!",
			"kubernetes: Int!",
			"openshift: Int!",
			"openshift4: Int!",
		}),
	)
}

type clusterCountByPlatformTypeResolver struct {
	ctx  context.Context
	root *Resolver
	data platformcve.ClusterCountByPlatformType
}

func (resolver *Resolver) wrapClusterCountByPlatformTypeWithContext(ctx context.Context, value platformcve.ClusterCountByPlatformType, err error) (*clusterCountByPlatformTypeResolver, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &clusterCountByPlatformTypeResolver{ctx: ctx, root: resolver, data: platformcve.NewEmptyClusterCountByPlatformType()}, nil
	}
	return &clusterCountByPlatformTypeResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Generic returns number of clusters of type GENERIC
func (resolver *clusterCountByPlatformTypeResolver) Generic(_ context.Context) int32 {
	return int32(resolver.data.GetGenericClusterCount())
}

// Kubernetes returns the number of clusters of type KUBERNETES
func (resolver *clusterCountByPlatformTypeResolver) Kubernetes(_ context.Context) int32 {
	return int32(resolver.data.GetKubernetesClusterCount())
}

// Openshift returns the number of clusters of type OPENSHIFT
func (resolver *clusterCountByPlatformTypeResolver) Openshift(_ context.Context) int32 {
	return int32(resolver.data.GetOpenshiftClusterCount())
}

// Openshift4 retruns the number of clusters of type OPENSHIFT4
func (resolver *clusterCountByPlatformTypeResolver) Openshift4(_ context.Context) int32 {
	return int32(resolver.data.GetOpenshift4ClusterCount())
}
