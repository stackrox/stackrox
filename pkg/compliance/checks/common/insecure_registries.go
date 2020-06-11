package common

import (
	"net"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
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

func insecureRegistries(info *compliance.ContainerRuntimeInfo) []*storage.ComplianceResultValue_Evidence {
	if info == nil {
		return FailList("No container runtime information is available to determine insecure registry configs")
	}

	var failed bool
	var results []*storage.ComplianceResultValue_Evidence
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
			results = append(results, Failf("Insecure registry with CIDR %q is configured", cidrStr))
		}
	}
	for _, registry := range info.GetInsecureRegistries().GetInsecureRegistries() {
		results = append(results, Failf("Insecure registry %q configured", registry))
		failed = true
	}

	if !failed {
		results = append(results, Pass("No insecure registries in public networks are configured"))
	}
	return results
}

// CheckNoInsecureRegistries checks that only registries in private subnets are configured as insecure.
func CheckNoInsecureRegistries(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
	return insecureRegistries(complianceData.ContainerRuntimeInfo)
}
