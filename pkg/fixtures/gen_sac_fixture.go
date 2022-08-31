package fixtures

import (
	"github.com/mauricelam/genny/generic"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// Resource represents a generic type that we use in the function below.
//
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=deployment_$GOFILE gen "Resource=*storage.Deployment"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=indicator_$GOFILE gen "Resource=*storage.ProcessIndicator"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=namespace_$GOFILE gen "Resource=*storage.NamespaceMetadata"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=pod_$GOFILE gen "Resource=*storage.Pod"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=role_bindings_$GOFILE gen "Resource=*storage.K8SRoleBinding"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=role_$GOFILE gen "Resource=*storage.K8SRole"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=service_account_$GOFILE gen "Resource=*storage.ServiceAccount"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=risk_$GOFILE gen "Resource=*storage.Risk"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=process_baseline_result_$GOFILE gen "Resource=*storage.ProcessBaselineResults"
//go:generate genny -in=$GOFILE -imp=github.com/stackrox/rox/generated/storage -out=process_baseline_$GOFILE gen "Resource=*storage.ProcessBaseline"
type Resource generic.Type

// GetSACTestResourceSet returns a set of mock Resource that can be used
// for scoped access control sets.
// It will include:
// 9 Resource scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 Resource scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 Resource scoped to Cluster3, 3 to each Namespace A / B / C.
func GetSACTestResourceSet(scopedResourceCreator func(id string, clusterID string, namespace string) Resource) []Resource {
	clusters := []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3}
	namespaces := []string{testconsts.NamespaceA, testconsts.NamespaceB, testconsts.NamespaceC}
	const numberOfAccounts = 3
	resources := make([]Resource, 0, len(clusters)*len(namespaces)*numberOfAccounts)
	for _, cluster := range clusters {
		for _, namespace := range namespaces {
			for i := 0; i < numberOfAccounts; i++ {
				resources = append(resources, scopedResourceCreator(uuid.NewV4().String(), cluster, namespace))
			}
		}
	}
	return resources
}
