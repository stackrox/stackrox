package fixtures

import (
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetSACTestResourceSet returns a set of mock Resource that can be used
// for scoped access control sets.
// It will include:
// 9 Resource scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 Resource scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 Resource scoped to Cluster3, 3 to each Namespace A / B / C.
func GetSACTestResourceSet[R any](scopedResourceCreator func(id string, clusterID string, namespace string) R) []R {
	clusters := []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3}
	namespaces := []string{testconsts.NamespaceA, testconsts.NamespaceB, testconsts.NamespaceC}
	const numberOfAccounts = 3
	resources := make([]R, 0, len(clusters)*len(namespaces)*numberOfAccounts)
	for _, cluster := range clusters {
		for _, namespace := range namespaces {
			for i := 0; i < numberOfAccounts; i++ {
				resources = append(resources, scopedResourceCreator(uuid.NewV4().String(), cluster, namespace))
			}
		}
	}
	return resources
}
