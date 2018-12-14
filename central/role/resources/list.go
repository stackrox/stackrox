// Package resources lists all resource types used by Central.
package resources

import (
	"sort"

	"github.com/stackrox/rox/pkg/auth/permissions"
)

// All resource types that we want to define (for the purposes of enforcing
// API permissions) must be defined here.
var (
	APIToken          = newResource("APIToken")
	Alert             = newResource("Alert")
	AuthProvider      = newResource("AuthProvider")
	Benchmark         = newResource("Benchmark")
	BenchmarkScan     = newResource("BenchmarkScan")
	BenchmarkSchedule = newResource("BenchmarkSchedule")
	BenchmarkTrigger  = newResource("BenchmarkTrigger")
	Cluster           = newResource("Cluster")
	DebugMetrics      = newResource("DebugMetrics")
	DebugLogs         = newResource("DebugLogs")
	Deployment        = newResource("Deployment")
	Detection         = newResource("Detection")
	Group             = newResource("Group")
	Image             = newResource("Image")
	ImageIntegration  = newResource("ImageIntegration")
	ImbuedLogs        = newResource("ImbuedLogs")
	Indicator         = newResource("Indicator")
	Node              = newResource("Node")
	Notifier          = newResource("Notifier")
	NetworkPolicy     = newResource("NetworkPolicy")
	NetworkGraph      = newResource("NetworkGraph")
	Policy            = newResource("Policy")
	Role              = newResource("Role")
	Secret            = newResource("Secret")
	ServiceIdentity   = newResource("ServiceIdentity")
	User              = newResource("User")

	allResources = make(map[permissions.Resource]struct{})
)

func newResource(name permissions.Resource) permissions.Resource {
	allResources[name] = struct{}{}
	return name
}

// ListAll returns a list of all resources.
func ListAll() []permissions.Resource {
	lst := make([]permissions.Resource, 0, len(allResources))
	for r := range allResources {
		lst = append(lst, r)
	}
	sort.Slice(lst, func(i, j int) bool {
		return lst[i] < lst[j]
	})
	return lst
}
