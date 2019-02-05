package common

import (
	"net"

	"github.com/docker/docker/api/types"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/netutil"
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

func insecureRegistries(ctx framework.ComplianceContext, info types.Info) {
	if info.RegistryConfig == nil {
		framework.PassNow(ctx, "No insecure registries are configured")
	}
	var failed bool
	for _, registry := range info.RegistryConfig.InsecureRegistryCIDRs {
		isPrivate := false
		for _, privateSubnet := range privateSubnets {
			if netutil.IsIPNetSubset(privateSubnet, (*net.IPNet)(registry)) {
				isPrivate = true
				break
			}
		}
		if !isPrivate {
			failed = true
			framework.Failf(ctx, "Insecure registry with CIDR %q is configured", registry.IP)
		}
	}
	for indexName, indexConfig := range info.RegistryConfig.IndexConfigs {
		if !indexConfig.Secure {
			failed = true
			framework.Failf(ctx, "Insecure registry %q is configured", indexName)
		}
	}
	if !failed {
		framework.Pass(ctx, "Docker is not running with insecure registries")
	}
}

// CheckNoInsecureRegistries checks that only registries in private subnets are configured as insecure.
func CheckNoInsecureRegistries(ctx framework.ComplianceContext) {
	PerNodeCheckWithDockerData(func(ctx framework.ComplianceContext, data *docker.Data) {
		insecureRegistries(ctx, data.Info)
	})(ctx)
}
