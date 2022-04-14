package k8sutil

import (
	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	openshiftAppsScheme "github.com/openshift/client-go/apps/clientset/versioned/scheme"
	openshiftAuthorizationScheme "github.com/openshift/client-go/authorization/clientset/versioned/scheme"
	openshiftBuildScheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	openshiftConfigScheme "github.com/openshift/client-go/config/clientset/versioned/scheme"
	openshiftConsoleScheme "github.com/openshift/client-go/console/clientset/versioned/scheme"
	openshiftImageScheme "github.com/openshift/client-go/image/clientset/versioned/scheme"
	openshiftImageRegistryScheme "github.com/openshift/client-go/imageregistry/clientset/versioned/scheme"
	openshiftNetworkScheme "github.com/openshift/client-go/network/clientset/versioned/scheme"
	openshiftOAuthScheme "github.com/openshift/client-go/oauth/clientset/versioned/scheme"
	openshiftOperatorScheme "github.com/openshift/client-go/operator/clientset/versioned/scheme"
	openshiftProjectScheme "github.com/openshift/client-go/project/clientset/versioned/scheme"
	openshiftQuotaScheme "github.com/openshift/client-go/quota/clientset/versioned/scheme"
	openshiftRouteScheme "github.com/openshift/client-go/route/clientset/versioned/scheme"
	openshiftSecurityScheme "github.com/openshift/client-go/security/clientset/versioned/scheme"
	openshiftServiceCertSignerScheme "github.com/openshift/client-go/servicecertsigner/clientset/versioned/scheme"
	openshiftTemplateScheme "github.com/openshift/client-go/template/clientset/versioned/scheme"
	openshiftUserScheme "github.com/openshift/client-go/user/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultGroupName = ""
)

var (
	// This will ensure that we also support old DeploymentConfigs which are not in the "apps.openshift.io/v1" group
	// but rather within the core group (for more info see: https://issues.redhat.com/browse/ROX-7515)
	legacyOpenshiftAppsGV = schema.GroupVersion{Group: defaultGroupName, Version: "v1"}
	schemeBuilder         = runtime.NewSchemeBuilder(addKnownTypes)

	openshiftSchemesToRegister = []addToScheme{
		openshiftAppsScheme.AddToScheme,
		openshiftAuthorizationScheme.AddToScheme,
		openshiftBuildScheme.AddToScheme,
		openshiftConfigScheme.AddToScheme,
		openshiftConsoleScheme.AddToScheme,
		openshiftImageScheme.AddToScheme,
		openshiftImageRegistryScheme.AddToScheme,
		openshiftNetworkScheme.AddToScheme,
		openshiftOAuthScheme.AddToScheme,
		openshiftOperatorScheme.AddToScheme,
		openshiftProjectScheme.AddToScheme,
		openshiftQuotaScheme.AddToScheme,
		openshiftRouteScheme.AddToScheme,
		openshiftSecurityScheme.AddToScheme,
		openshiftServiceCertSignerScheme.AddToScheme,
		openshiftTemplateScheme.AddToScheme,
		openshiftUserScheme.AddToScheme,
		schemeBuilder.AddToScheme,
	}
)

type addToScheme = func(s *runtime.Scheme) error

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

// AddOpenShiftSchemes will add openshift schemes listed within https://github.com/openshift/client-go to the
// runtime.Scheme.
// It will return any error that may occur during adding of a scheme.
func AddOpenShiftSchemes(scheme *runtime.Scheme) error {
	for _, addScheme := range openshiftSchemesToRegister {
		if err := addScheme(scheme); err != nil {
			return err
		}
	}
	return nil
}
