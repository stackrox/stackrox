// Package resources lists all resource types used by Central.
package resources

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// All resource types that we want to define (for the purposes of enforcing
// API permissions) must be defined here.
var (
	APIToken              = newResource("APIToken")
	Alert                 = newResource("Alert")
	AuthProvider          = newResource("AuthProvider")
	Benchmark             = newResource("Benchmark")
	BenchmarkScan         = newResource("BenchmarkScan")
	BenchmarkSchedule     = newResource("BenchmarkSchedule")
	BenchmarkTrigger      = newResource("BenchmarkTrigger")
	Cluster               = newResource("Cluster")
	Compliance            = newResource("Compliance")
	ComplianceRunSchedule = newResource("ComplianceRunSchedule")
	ComplianceRuns        = newResource("ComplianceRuns")
	DebugMetrics          = newResource("DebugMetrics")
	DebugLogs             = newResource("DebugLogs")
	Deployment            = newResource("Deployment")
	Detection             = newResource("Detection")
	Group                 = newResource("Group")
	Image                 = newResource("Image")
	ImageIntegration      = newResource("ImageIntegration")
	ImbuedLogs            = newResource("ImbuedLogs")
	Indicator             = newResource("Indicator")
	Namespace             = newResource("Namespace")
	Node                  = newResource("Node")
	Notifier              = newResource("Notifier")
	NetworkPolicy         = newResource("NetworkPolicy")
	NetworkGraph          = newResource("NetworkGraph")
	Policy                = newResource("Policy")
	Role                  = newResource("Role")
	Secret                = newResource("Secret")
	ServiceIdentity       = newResource("ServiceIdentity")
	User                  = newResource("User")

	allResources = permissions.NewResourceSet()
)

func newResource(name permissions.Resource) permissions.Resource {
	allResources.Add(name)
	return name
}

// ListAll returns a list of all resources.
func ListAll() []permissions.Resource {
	return allResources.AsSortedSlice(func(i, j permissions.Resource) bool {
		return i < j
	})
}
