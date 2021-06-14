package listener

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

const (
	complianceGroup           = "compliance.openshift.io"
	complianceVersion         = "v1alpha1"
	complianceGroupVersion    = complianceGroup + "/" + complianceVersion
	complianceCheckResultName = "compliancecheckresults"
)

var (
	complianceGVR = schema.GroupVersionResource{
		Group:    complianceGroup,
		Version:  complianceVersion,
		Resource: complianceCheckResultName,
	}

	profileGVR = schema.GroupVersionResource{
		Group:    complianceGroup,
		Version:  complianceVersion,
		Resource: "profiles",
	}

	scanSettingBindingGVR = schema.GroupVersionResource{
		Group:    complianceGroup,
		Version:  complianceVersion,
		Resource: "scansettingbindings",
	}

	ruleGVR = schema.GroupVersionResource{
		Group:    complianceGroup,
		Version:  complianceVersion,
		Resource: "rules",
	}
)

func complianceCRDExists(client kubernetes.Interface) (bool, error) {
	resourceList, err := client.Discovery().ServerResourcesForGroupVersion(complianceGroupVersion)
	if err != nil {
		return false, err
	}
	for _, apiResource := range resourceList.APIResources {
		if apiResource.Name == complianceCheckResultName {
			return true, nil
		}
	}
	return false, nil
}
