package k8sutil

import (
	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultGroupName = ""
)

var (
	legacyOpenshiftAppsGV = schema.GroupVersion{Group: defaultGroupName, Version: "v1"}

	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddLegacyOpenshiftAppsToScheme provides legacy Openshift Apps schemes which use the default group instead of apps.openshift.io
	AddLegacyOpenshiftAppsToScheme = schemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	// Legacy Openshift config (DeploymentConfig) is not using the
	// apps.openshift.io API group, but rather the default group name
	// Providing a legacy scheme builder to add this legacy group to runtime.Scheme
	scheme.AddKnownTypes(legacyOpenshiftAppsGV,
		&openshiftAppsV1.DeploymentConfig{},
		&openshiftAppsV1.DeploymentConfigList{},
		&openshiftAppsV1.DeploymentConfigRollback{},
		&openshiftAppsV1.DeploymentRequest{},
		&openshiftAppsV1.DeploymentLog{},
		&openshiftAppsV1.DeploymentLogOptions{},
	)
	metav1.AddToGroupVersion(scheme, legacyOpenshiftAppsGV)
	return nil
}
