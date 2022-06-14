package listener

import (
	"github.com/stackrox/stackrox/pkg/complianceoperator"
	"k8s.io/client-go/kubernetes"
)

func complianceCRDExists(client kubernetes.Interface) (bool, error) {
	resourceList, err := client.Discovery().ServerResourcesForGroupVersion(complianceoperator.ComplianceGroupVersion)
	if err != nil {
		return false, err
	}
	for _, apiResource := range resourceList.APIResources {
		if apiResource.Name == complianceoperator.CheckResultGVR.Resource {
			return true, nil
		}
	}
	return false, nil
}
