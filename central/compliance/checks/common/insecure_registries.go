package common

import (
	"net"

	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/pkg/netutil"
)

var (
	privateSubnets = []*net.IPNet{
		netutil.MustParseCIDR("127.0.0.0/8"),    // IPv4 localhost
		netutil.MustParseCIDR("10.0.0.0/8"),     // class A
		netutil.MustParseCIDR("172.16.0.0/12"),  // class B
		netutil.MustParseCIDR("192.168.0.0/16"), // class C
		netutil.MustParseCIDR("::1/128"),        // IPv6 localhost
		netutil.MustParseCIDR("fd00::/8"),       // IPv6 ULA
	}
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
			for _, privateSubnets := range privateSubnets {
				if netutil.IsIPNetSubset(privateSubnets, cidr) {
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
