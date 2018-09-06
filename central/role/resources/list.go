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
	Deployment        = newResource("Deployment")
	Detection         = newResource("Detection")
	DNRIntegration    = newResource("DNRIntegration")
	Image             = newResource("Image")
	ImageIntegration  = newResource("ImageIntegration")
	ImbuedLogs        = newResource("ImbuedLogs")
	Indicators        = newResource("Indicator")
	Notifier          = newResource("Notifier")
	NetworkPolicy     = newResource("NetworkPolicy")
	Policy            = newResource("Policy")
	Role              = newResource("Role")
	Secret            = newResource("Secret")
	ServiceIdentity   = newResource("ServiceIdentity")

	allResources = make(map[permissions.Resource]struct{})
)

func newResource(name string) permissions.Resource {
	r := permissions.Resource(name)
	allResources[r] = struct{}{}
	return r
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
