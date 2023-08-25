package common

import (
	"net"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/netutil"
)

func insecureRegistries(ctx framework.ComplianceContext, info *compliance.ContainerRuntimeInfo) {
	if info == nil {
		framework.Fail(ctx, "No container runtime information is available to determine insecure registry configs")
		return
	}

	var failed bool
	for _, cidrStr := range info.GetInsecureRegistries().GetInsecureCidrs() {
		_, cidr, _ := net.ParseCIDR(cidrStr)
		isPrivate := false
		if cidr != nil {
			for _, privateSubnet := range netutil.GetPrivateSubnets() {
				if netutil.IsIPNetSubset(privateSubnet, cidr) {
					isPrivate = true
					break
				}
			}
		}
		if !isPrivate {
			failed = true
			framework.Failf(ctx, "Insecure registry with CIDR %q is configured", cidrStr)
		}
	}
	for _, registry := range info.GetInsecureRegistries().GetInsecureRegistries() {
		framework.Failf(ctx, "Insecure registry %q configured", registry)
		failed = true
	}

	if !failed {
		framework.Pass(ctx, "No insecure registries in public networks are configured")
	}
}

// CheckNoInsecureRegistries checks that only registries in private subnets are configured as insecure.
func CheckNoInsecureRegistries(ctx framework.ComplianceContext) {
	PerNodeCheck(func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
		insecureRegistries(ctx, ret.GetContainerRuntimeInfo())
	})(ctx)
}
