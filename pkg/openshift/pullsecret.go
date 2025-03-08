package openshift

import "github.com/stackrox/rox/generated/storage"

const (
	// GlobalPullSecretNamespace contains the namespace name where the global pull secret exists.
	//#nosec G101 -- This is a false positive
	GlobalPullSecretNamespace = "openshift-config"

	// GlobalPullSecretName contains the name of the OCP global pull secret.
	GlobalPullSecretName = "pull-secret"
)

// GlobalPullSecret returns true if secret namespace and name represent the
// OCP global pull secret.
func GlobalPullSecret(secretNamespace, secretName string) bool {
	return secretNamespace == GlobalPullSecretNamespace && secretName == GlobalPullSecretName
}

// GlobalPullSecretIntegration returns true if the integration has a source and
// that source indicates it is from the OCP global pull secret.
func GlobalPullSecretIntegration(integration *storage.ImageIntegration) bool {
	if integration == nil {
		return false
	}

	source := integration.GetSource()
	if source == nil {
		return false
	}

	return GlobalPullSecret(source.GetNamespace(), source.GetImagePullSecretName())
}
