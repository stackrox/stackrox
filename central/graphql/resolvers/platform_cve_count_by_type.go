package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/views/platformcve"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlatformCVECountByType", []string{
			"kubernetes: Int!",
			"openshift: Int!",
			"istio: Int!",
		}),
	)
}

type platformCVECountByTypeResolver struct {
	ctx  context.Context
	root *Resolver
	data platformcve.CVECountByType
}

func (resolver *Resolver) wrapPlatformCVECountByTypeWithContext(ctx context.Context, value platformcve.CVECountByType, err error) (*platformCVECountByTypeResolver, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return &platformCVECountByTypeResolver{ctx: ctx, root: resolver, data: platformcve.NewEmptyCVECountByType()}, nil
	}
	return &platformCVECountByTypeResolver{ctx: ctx, root: resolver, data: value}, nil
}

// Kubernetes returns the number of platform CVEs of type K8S_CVE
func (resolver *platformCVECountByTypeResolver) Kubernetes(_ context.Context) int32 {
	return int32(resolver.data.GetKubernetesCVECount())
}

// Openshift returns the number of platform CVEs of type OPENSHIFT_CVE
func (resolver *platformCVECountByTypeResolver) Openshift(_ context.Context) int32 {
	return int32(resolver.data.GetOpenshiftCVECount())
}

// Istio returns the number of platform CVEs of type ISTIO_CVE
func (resolver *platformCVECountByTypeResolver) Istio(_ context.Context) int32 {
	return int32(resolver.data.GetIstioCVECount())
}
